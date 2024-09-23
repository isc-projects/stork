"""
Collects the performance metrics of the services managed by supervisor.

Inspired by Superlance project: https://github.com/Supervisor/superlance
"""

from datetime import timedelta
import os
import time
import subprocess
import sys
from xmlrpc.client import ServerProxy


class PerformanceMetricsCollector:
    """
    Class for collecting the performance metrics.

    The collected samples are stored in the separate lines. The line format is:
    <timestamp><tab><service name><tab><counter name><tab><counter value>

    """

    def __init__(self, output_path, interval=timedelta(seconds=1), rpc=None):
        """Requires the path to the file where the metrics will be stored."""
        self.output_path = output_path
        self.interval = interval

        self.rpc = rpc
        if rpc is None:
            self.rpc = ServerProxy("http://127.0.0.1:9001/RPC2")

    def run_forever(self):
        """Collects the metrics until terminated."""
        interval_seconds = self.interval.total_seconds()
        start_time = time.monotonic()
        while True:
            self.handle_tick()
            time.sleep(
                interval_seconds - (time.monotonic() - start_time) % interval_seconds
            )

    def handle_tick(self):
        """Reads the performance counters of the supervisor services and
        writes them to the output file."""

        # The event frequency is relatively long so it should be enough to just
        # pick the current time. In fact, it isn't an exact time when the data
        # is collected.
        timestamp = time.time()
        # Fetch the process details from the system.
        process_details = self.fetch_process_details()
        # Current process PID.
        this_pid = os.getpid()

        # List the supervisor services.
        infos = self.rpc.supervisor.getAllProcessInfo()

        # Rotate the performance metrics file when it is too large to prevent
        # the file from growing indefinitely for long-running services
        # (e.g. demo)
        if os.path.exists(self.output_path):
            size = os.path.getsize(self.output_path)
            if size > 1024 * 1024 * 10:  # 10 MB
                os.rename(self.output_path, f"{self.output_path}.old")

        # Append the performance metrics.
        with open(self.output_path, "a", encoding="utf-8") as f:
            for info in infos:
                pid = info["pid"]
                name = info["name"]

                if not pid:
                    # The service is not running.
                    continue

                if pid == this_pid:
                    # The process is the collector itself.
                    continue

                # Sum the process counters with the counters of its children.
                counters = self.cumulate_counters(process_details, pid)
                # Write the counters to the file.
                for counter_name, counter_value in counters.items():
                    f.write(f"{timestamp}\t{name}\t{counter_name}\t{counter_value}\n")

    def cumulate_counters(self, data, pid):
        """Sums the counters of the process with the counters of its children."""

        def find_children(pid):
            """Recursively finds the children of the process."""
            children = []
            for p in data:
                if data[p]["ppid"] == pid:
                    children.append(p)
                    children.extend(find_children(p))
            return children

        family_pids = [
            pid,
        ] + find_children(pid)
        counters = {}

        for family_pid in family_pids:
            child_counters = data[family_pid]["counters"]
            for counter in child_counters:
                if counter not in counters:
                    counters[counter] = 0
                counters[counter] += child_counters[counter]
        return counters

    def fetch_process_details(self):
        """
        Executes the ps system command and parses its output.
        Returns the dictionary where the keys are the process IDs and the
        values are the dictionaries with the following keys:
        - ppid: the parent process ID,
        - counters: the dictionary where the keys are the counter names and
            the values are the counter values.
        """
        cmd = ["ps", "ax", "-o", "pid= ppid= pcpu= pmem= rss= vsz="]
        cmd_output = subprocess.check_output(cmd).decode("utf-8")

        data = {}

        for line in cmd_output.splitlines():
            pid, ppid, cpu, mem, rss, vsz = line.split()
            pid = int(pid)
            ppid = int(ppid)
            cpu = float(cpu)
            mem = float(mem)
            rss = int(rss) * 1024.0  # RSS is in KB. Convert to bytes.
            vsz = int(vsz) * 1024.0  # VSZ is in KB. Convert to bytes.

            data[pid] = {
                "ppid": ppid,
                "counters": {
                    "cpu [%]": cpu,
                    "mem [%]": mem,
                    "rss [B]": rss,
                    "vsz [B]": vsz,
                },
            }
        return data


def main():
    """Runs the collector."""
    collector = PerformanceMetricsCollector(
        output_path=sys.argv[1],
        interval=timedelta(
            milliseconds=int(sys.argv[2]) if len(sys.argv) > 2 else 1000
        ),
    )
    collector.run_forever()


if __name__ == "__main__":
    main()
