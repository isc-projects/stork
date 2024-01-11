'''
Summarizes the performance reports collected from the containers and presents
the data on charts.

The performance report format is described in the `PerformanceMetricsCollector`.
Single test run may produce multiple reports.
'''
from typing import List, Set, Dict
from collections import namedtuple

import plotly
import plotly.subplots


ReportEntry = namedtuple(
    'ReportEntry',
    ['timestamp', 'service_name', 'counter_name', 'counter_value']
)
ReportEntries = List[ReportEntry]


def read_performance_report(path: str) -> ReportEntries:
    '''Reads the entries from the provided path.'''
    entries = []
    with open(path, 'r', encoding='utf-8') as f:
        lines = f.readlines()
        for line in lines:
            line = line.strip()
            if not line:
                continue
            timestamp, service_name, counter_name, counter_value = line.split('\t')
            entry = ReportEntry(float(timestamp), service_name, counter_name, float(counter_value))
            entries.append(entry)
    return entries


def list_counter_names(entries: ReportEntries) -> Set[str]:
    '''Returns a list of all counter names.'''
    counter_names = set()
    for entry in entries:
        counter_names.add(entry.counter_name)
    return counter_names


def list_service_names(entries: ReportEntries) -> Set[str]:
    '''Returns a list of all service names.'''
    service_names = set()
    for entry in entries:
        service_names.add(entry.service_name)
    return service_names


def get_min_timestamp(entries: ReportEntries) -> float:
    '''Returns the minimum timestamp from the provided entries.'''
    min_timestamp = None
    for entry in entries:
        if min_timestamp is None or entry.timestamp < min_timestamp:
            min_timestamp = entry.timestamp
    return min_timestamp


def group_by_service(entries: ReportEntries) -> Dict[str, ReportEntries]:
    '''Groups the performance report entries by the service name.'''
    groups = {}
    for entry in entries:
        service_name = entry.service_name
        if service_name not in groups:
            groups[service_name] = []
        groups[service_name].append(entry)
    return groups


def group_by_counter(entries: ReportEntries) -> Dict[str, ReportEntries]:
    '''Groups the performance report entries by the counter name.'''
    groups = {}
    for entry in entries:
        counter_name = entry.counter_name
        if counter_name not in groups:
            groups[counter_name] = []
        groups[counter_name].append(entry)
    return groups


def plot_reports(report_paths: List[str], output_path: str = None):
    '''
    Function reads the provided performance reports and plots the data on
    charts. There are charts grouped by metric name (a series per service).
    '''
    # Read entries from all reports.
    entries = [e for path in report_paths for e in read_performance_report(path)]

    # Get the minimum timestamp.
    min_timestamp = get_min_timestamp(entries)

    # Group the entries by counter name.
    counter_groups = group_by_counter(entries)

    # Prepare the figure.
    fig = plotly.subplots.make_subplots(
        rows=len(counter_groups),
        cols=1,
        subplot_titles=list(counter_groups.keys()),
        shared_xaxes=True,
        vertical_spacing=0.04,
        x_title='Time (s)',
        column_titles=["Performance metrics<br>"+"<br>".join(report_paths)+"<br>&nbsp;"]
    )

    # Choose color palette.
    colors = plotly.colors.qualitative.D3

    # Generate a chart for each counter.
    for cix, (_, counter_entries) in enumerate(counter_groups.items(), 1):
        # Generate a series for each service.
        for six, (service_name, service_entries) in enumerate(group_by_service(counter_entries).items()):
            # Sort the entries by timestamp.
            service_entries.sort(key=lambda e: e.timestamp)

            fig.add_scatter(
                x=[e.timestamp - min_timestamp for e in service_entries],
                y=[e.counter_value for e in service_entries],
                name=service_name,
                row=cix,
                col=1,
                legendgroup='group1',
                showlegend=cix == 1,
                line_color=colors[six % len(colors)],
            )

    # Show the figure.
    if output_path is None:
        fig.show()
    else:
        fig.write_html(output_path)


if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser(
        description='Summarizes the performance reports collected from the '
                    'containers and presents the data on charts.')
    parser.add_argument('reports', metavar='REPORT', type=str, nargs='+',
                        help='a path to a performance report')
    parser.add_argument('-o', '--output', metavar='OUTPUT', type=str,
                        help='a path to the output file', default=None)
    args = parser.parse_args()

    plot_reports(args.reports, args.output)