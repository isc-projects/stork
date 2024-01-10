'''
Collects the performance metrics of the services managed by supervisor.

Inspired by Superlance project: https://github.com/Supervisor/superlance
'''

import time
import subprocess
import sys
from xmlrpc.client import ServerProxy


def write_stderr(s):
    sys.stderr.write(s)
    sys.stderr.flush()


class EventListenerProtocol:
    '''
    Defines the methods to communicate with supervisor.
    '''
    def __init__(self, stdin=sys.stdin, stdout=sys.stdout):
        '''
        Accepts the stdin and stdout file-like objects.
        The default values are sys.stdin and sys.stdout. They may be overriden
        for testing purposes.
        '''
        self.stdin = stdin
        self.stdout = stdout

    def write_stdout(self, s):
        '''
        Write a string to stdout and flush it.
        Noe, only eventlistener protocol messages may be sent to stdout
        '''
        self.stdout.write(s)
        self.stdout.flush()

    def send_ok(self):
        '''
        Transition from READY to ACKNOWLEDGED state.
        The format of the message is:
        RESULT <number of the content bytes><new line><content string>
        '''
        self.write_stdout('RESULT 2\nOK')

    def send_ready(self):
        '''Transition from ACKNOWLEDGED to READY state.'''
        self.write_stdout('READY\n')

    @staticmethod
    def _parse_headers(line):
        '''Parse the headers from supervisor.'''
        headers = dict([ x.split(':') for x in line.split() ])
        return headers

    def read_event_data(self):
        '''Read the event data from supervisor. Parses the headers and reads
        the data.'''
        header_line = self.stdin.readline()
        headers = EventListenerProtocol._parse_headers(header_line)
        data = sys.stdin.read(int(headers['len']))
        return headers, data


class PerformanceMetricsCollector:
    '''
    Class for performing the performance metrics collection.

    The collected samples are stored in the separate lines. The line format is:
    <timestamp><tab><service name><tab><counter name><tab><counter value>

    '''
    def __init__(self, output_path, protocol=None, rpc=None):
        '''Requires the path to the file where the metrics will be stored.'''
        self.output_path = output_path
        self.protocol = protocol if protocol else EventListenerProtocol()

        self.rpc = rpc
        if rpc is None:
            self.rpc = ServerProxy("http://127.0.0.1:9001/RPC2")

    def run_forever(self):
        '''Collects the metrics until terminated.'''
        while 1:
            self.handle_next_event()

    def handle_next_event(self):
        '''Waits for, reads the next supervisor, and processes it.'''
        # Transition from ACKNOWLEDGED to READY.
        self.protocol.send_ready()

        write_stderr("HELLO LOOP!")

        # Read event payload.
        headers, _ = self.protocol.read_event_data()

        if not headers['eventname'].startswith('TICK'):
            # Do nothing with non-TICK events.
            self.protocol.send_ok()

        # Collect the metrics.
        self.handle_tick()

        # Transition from READY to ACKNOWLEDGED.
        self.protocol.send_ok()

    def handle_tick(self):
        '''Reads the performance counters of the supervisor services and
        writes them to the output file.'''
        write_stderr("HELLO TICK!")

        # The event frequency is 5 second so it should be enough to just pick
        # the current time. In fact, it isn't an exact time when the data is
        # collected.
        timestamp = time.time()
        # Fetch the process details from the system.
        processDetails = self.fetch_process_details()

        # List the supervisor services.
        infos = self.rpc.supervisor.getAllProcessInfo()

        # Append the performance metrics.
        with open(self.output_path, 'a', encoding='utf-8') as f:
            for info in infos:
                pid = info['pid']
                name = info['name']

                if not pid:
                    # The service is not running.
                    continue

                # Sum the process counters with the counters of its children.
                counters = self.cumulate_counters(processDetails, pid)
                # Write the counters to the file.
                for counter_name, counter_value in counters.items():
                    f.write(f'{timestamp}\t{name}\t{counter_name}\t{counter_value}\n')  

    def cumulate_counters(self, data, pid):
        '''Sums the counters of the process with the counters of its children.'''

        def find_children(pid):
            '''Recursively finds the children of the process.'''
            children = []
            for p in data:
                if data[p]['ppid'] == pid:
                    children.append(p)
                    children.extend(find_children(p))
            return children

        children = find_children(pid)
        counters = {}

        for child in children:
            child_counters = data[child]['counters']
            for counter in child_counters:
                if counter not in counters:
                    counters[counter] = 0
                counters[counter] += child_counters[counter]
        return counters

    def fetch_process_details(self):
        '''
        Executes the ps system command and parses its output.
        Returns the dictionary where the keys are the process IDs and the
        values are the dictionaries with the following keys:
        - ppid: the parent process ID,
        - counters: the dictionary where the keys are the counter names and
            the values are the counter values.
        '''
        cmd = ['ps', 'ax', '-o', "pid= ppid= pcpu= pmem= rss="]
        cmd_output = subprocess.check_output(cmd).decode('utf-8')

        data = {}

        for line in cmd_output.splitlines():
            pid, ppid, cpu, mem, rss = line.split()
            pid = int(pid)
            ppid = int(ppid)
            cpu = float(cpu)
            mem = float(mem)
            rss = int(rss)
            
            data[pid] = {
                'ppid': ppid,
                'counters': {
                    'cpu': cpu,
                    'mem': mem,
                    'rss': rss
                }
            }
        return data





def main():
    write_stderr("HELLO WORLD!")
    collector = PerformanceMetricsCollector(sys.argv[1])
    collector.run_forever()


if __name__ == '__main__':
    main()