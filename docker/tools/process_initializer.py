'''
The supervisorD event listener for initializing the application instances.
It waits until the application process is running, the executes all the scripts
in the provided directory prefixed by the process name. The scripts remember if
it was already executed for a particular application instance and if it was, it
will not execute again.

The scripts directory must contain a subdirectory for application processes,
and each subdirectory must contain the initialization scripts for that process.

For troubleshooting, the event listener writes logs to stderr, which are then
saved by supervisorD in a separate log file: /var/log/supervisor/process_initializer.err.log
They are not included in the main supervisorD logs until the DEBUG log level is
enabled in the [supervisord] section of the supervisord.conf file.
'''
import argparse
import os
import sys
from datetime import datetime
import subprocess


def prepare_cli_parser():
    '''Prepares the command line argument parser.'''
    parser = argparse.ArgumentParser(description='The supervisorD event listener for initializing the application instances.')
    parser.add_argument('-d', '--scripts-dir', type=str, required=True, help='The directory containing the initialization scripts.')
    parser.add_argument('-t', '--temp-dir', required=True, help='The temporary directory for storing the state of executed scripts.')
    return parser


def validate_args(args):
    '''Checks if the provided arguments are valid. It verifies all paths and
    permissions. If any check fails, it tries to fix it if possible.'''
    scripts_dir = args.scripts_dir
    temp_dir = args.temp_dir
    # Check if the directories exist and are accessible. If not, print an error
    # message and exit.
    if not os.path.isdir(scripts_dir):
        print(f"Error: The scripts directory '{scripts_dir}' does not exist or is not a directory.")
        sys.exit(1)
    if not os.path.exists(temp_dir):
        try:
            os.makedirs(temp_dir)
        except Exception as e:
            print(f"Error: Failed to create the temporary directory '{temp_dir}'. {e}")
            sys.exit(1)
    if not os.path.isdir(temp_dir):
        print(f"Error: The temporary directory '{temp_dir}' does not exist or is not a directory.")
        sys.exit(1)
    # The scripts directory must be readable and all files in it must be
    # executable.
    if not os.access(scripts_dir, os.R_OK):
        print(f"Error: The scripts directory '{scripts_dir}' is not readable.")
        sys.exit(1)
    for app_dir_name in os.listdir(scripts_dir):
        app_dir_path = os.path.join(scripts_dir, app_dir_name)
        if not os.path.isdir(app_dir_path):
            continue
        for script_name in os.listdir(app_dir_path):
            script_path = os.path.join(app_dir_path, script_name)
            if not os.path.isfile(script_path):
                continue
            if not os.access(script_path, os.X_OK):
                try:
                    os.chmod(script_path, 0o755)
                except Exception as e:
                     print(f"Error: The script '{script_path}' is not executable and failed to set executable permission. {e}")
                     sys.exit(1)
            # Check if the script has a shebang.
            with open(script_path, 'r') as f:
                first_line = f.readline()
                if not first_line.startswith("#!"):
                    print(f"Error: The script '{script_path}' does not have a shebang. Please add a shebang line to the script.")
                    sys.exit(1)
    # The temporary directory must be writable.
    if not os.access(temp_dir, os.W_OK):
        print(f"Error: The temporary directory '{temp_dir}' is not writable.")
        sys.exit(1)


def has_been_executed(temp_dir, process_name):
    '''Indicates if the scripts for the given process name have been executed.'''
    return os.path.exists(os.path.join(temp_dir, process_name))


def mark_as_executed(temp_dir, process_name):
    '''Creates a temporary but persistent file to indicate that the scripts for
    the given process name have been executed.'''
    now = datetime.now()
    with open(os.path.join(temp_dir, process_name), 'w') as f:
        f.write(now.isoformat())


def write_stdout(s):
    '''Writes a message to stdout. Only event listener protocol messages may be
    sent to stdout.'''
    sys.stdout.write(s)
    sys.stdout.flush()


def write_stderr(severity, message):
    '''
    Writes a message to stderr. This can be used for logging and debugging.
    Accepts a severity level (e.g., "INFO", "WARNING", "ERROR") and a message.
    The message is prefixed with a timestamp and the severity level for better
    readability. The messages written to stderr will not interfere with the
    communication with supervisorD, which uses stdout for event messages.
    '''
    s = f"{datetime.now().isoformat()} | {severity:<7} | {message}"
    sys.stderr.write(s)
    sys.stderr.flush()


def signal_ready():
    '''Sends a READY message to supervisorD to indicate that the event listener
    is ready to receive events.'''
    write_stdout("READY\n")


def write_response(success: bool):
    '''Sends a response to supervisorD to indicate the result of processing the
    event. The non-zero code indicates failure and will cause supervisorD to
    retry the event.'''
    if success:
        s = "RESULT 2\nOK"
    else: 
        s = "RESULT 4\nFAIL"
    write_stdout(s)

def execute_scripts(scripts_dir, process_name) -> bool:
    '''
    Executes the initialization scripts for the given process name.
    It is expected that the scripts have shebang and are executable.
    '''
    app_dir_path = os.path.join(scripts_dir, process_name)
    if not os.path.isdir(app_dir_path):
        return True
    write_stderr("INFO", f"Found initialization directory '{app_dir_path}' for process '{process_name}'.\n")
    for script_name in sorted(os.listdir(app_dir_path)):
        script_path = os.path.join(app_dir_path, script_name)
        if not os.path.isfile(script_path):
            continue
        if not os.access(script_path, os.X_OK):
            write_stderr("WARNING", f"The script '{script_path}' is not executable. Skipping.\n")
            continue
        write_stderr("INFO", f"Executing initialization script '{script_path}' for process '{process_name}'.\n")
        result = subprocess.run([script_path], capture_output=True, text=True)
        if result.returncode != 0:
            write_stderr("ERROR", f"The script '{script_path}' exited with code {result.returncode}. Stopping initialization.\n")
            write_stderr("ERROR", f"Output:\n{result.stdout}")
            write_stderr("ERROR", f"Error Output:\n{result.stderr}")
            return False
    return True

def single_iteration(scripts_dir, temp_dir) -> bool:
    '''
    Processes a single event from supervisorD. It reads the event from stdin
    check if the related process has already been initialized, and if not, it
    executes the initialization scripts for that process. It returns True if
    the processing was successful, or False if it failed.
    '''
    line = sys.stdin.readline()
    if not line:
        return False

    # Parse the event line.
    # See the event specification: https://supervisord.org/events.html#process-state-running-event-type
    # Example line: processname:process_initializer groupname:process_initializer from_state:STARTING pid:38ver:3.0 server:supervisor serial:7 pool:process_initializer poolserial:7 eventname:PROCESS_STATE_RUNNING len:68
    parts = line.split()
    data = {}
    for part in parts:
        key, value = part.split(":", 1)
        data[key] = value

    event_name = data.get("eventname")
    if event_name != "PROCESS_STATE_RUNNING":
        write_stderr("WARNING", f"Received unexpected event '{event_name}'. Ignoring.\n")
        return True
        
    process_name = data.get("processname")
    if not process_name:
        write_stderr("WARNING", f"Received event without process name. Ignoring: {line}.\n")
        return True
    if has_been_executed(temp_dir, process_name):
        write_stderr("INFO", f"Initialization scripts for process '{process_name}' have already been executed. Skipping.\n")
        return True
    ok = execute_scripts(scripts_dir, process_name)
    if ok:
        mark_as_executed(temp_dir, process_name)
        write_stderr("INFO", f"Successfully executed initialization scripts for process '{process_name}'.\n")
    else:
        write_stderr("ERROR", f"Failed to execute initialization scripts for process '{process_name}'.\n")
        # Do not mark as executed, so it will be retried on the next event.
    return ok


def event_loop(scripts_dir, temp_dir):
    '''
    The main event loop of the event listener. It signals that the listener is
    ready for a next event and ensures that the response is always sent back,
    even if an error occurs to not break the communication protocol.
    '''
    write_stderr("INFO", f"Starting the process initializer event loop\n")
    while True:
        signal_ready()
        try:
            ok = single_iteration(scripts_dir, temp_dir)
        except Exception as e:
            write_stderr("ERROR", f"An exception occurred during event processing. {e}\n")
            ok = False
        write_response(ok)


def main():
    '''The listener entry point. Parses the CLI arguments and starts the event
    loop.'''
    parser = prepare_cli_parser()
    args = parser.parse_args()
    validate_args(args)
    event_loop(args.scripts_dir, args.temp_dir)


if __name__ == "__main__":
    main()
