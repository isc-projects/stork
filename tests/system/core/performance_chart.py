"""
Summarizes the performance reports collected from the containers and presents
the data on charts.

The performance report format is described in the `PerformanceMetricsCollector`.
Single test run may produce multiple reports.
"""

from collections import namedtuple
import os.path
from typing import List, Set, Dict

import plotly
import plotly.subplots


ReportEntry = namedtuple(
    "ReportEntry",
    ["timestamp", "container", "service_name", "counter_name", "counter_value"],
)
ReportEntries = List[ReportEntry]


def read_performance_report(path: str) -> ReportEntries:
    """Reads the entries from the provided path."""
    container = os.path.basename(path)
    container, _ = os.path.splitext(container)

    entries = []
    with open(path, "r", encoding="utf-8") as f:
        lines = f.readlines()
        for line in lines:
            line = line.strip()
            if not line:
                continue
            timestamp, service_name, counter_name, counter_value = line.split("\t")
            entry = ReportEntry(
                float(timestamp),
                container,
                service_name,
                counter_name,
                float(counter_value),
            )
            entries.append(entry)
    return entries


def list_counter_names(entries: ReportEntries) -> Set[str]:
    """Returns a list of all counter names."""
    counter_names = set()
    for entry in entries:
        counter_names.add(entry.counter_name)
    return counter_names


def list_service_names(entries: ReportEntries) -> Set[str]:
    """Returns a list of all service names."""
    service_names = set()
    for entry in entries:
        service_names.add(entry.service_name)
    return service_names


def get_min_timestamp(entries: ReportEntries) -> float:
    """Returns the minimum timestamp from the provided entries."""
    min_timestamp = None
    for entry in entries:
        if min_timestamp is None or entry.timestamp < min_timestamp:
            min_timestamp = entry.timestamp
    return min_timestamp


def group_by_service(entries: ReportEntries) -> Dict[str, ReportEntries]:
    """Groups the performance report entries by the service name."""
    groups = {}
    for entry in entries:
        service_name = entry.service_name
        if service_name not in groups:
            groups[service_name] = []
        groups[service_name].append(entry)
    return groups


def group_by_container(entries: ReportEntries) -> Dict[str, ReportEntries]:
    """Groups the performance report entries by the container name."""
    groups = {}
    for entry in entries:
        container = entry.container
        if container not in groups:
            groups[container] = []
        groups[container].append(entry)
    return groups


def group_by_counter(entries: ReportEntries) -> Dict[str, ReportEntries]:
    """Groups the performance report entries by the counter name."""
    groups = {}
    for entry in entries:
        counter_name = entry.counter_name
        if counter_name not in groups:
            groups[counter_name] = []
        groups[counter_name].append(entry)
    return groups


# pylint: disable=too-many-locals
def plot_reports(report_paths: List[str], output_path: str = None):
    """
    Function reads the provided performance reports and plots the data on
    charts. There are charts grouped by metric name (a series per service).
    """
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
        x_title="Time (s)",
        column_titles=[
            "Performance metrics<br>"
            + "<br>".join(str(p) for p in report_paths)
            + "<br>&nbsp;"
        ],
    )

    # Choose color palette.
    colors = plotly.colors.qualitative.Dark24 + plotly.colors.qualitative.Alphabet
    color_map = {}
    color_idx = 0

    # Generate a chart for each counter.
    for cix, (_, counter_entries) in enumerate(counter_groups.items(), 1):
        # Generate a series for each service.
        service_groups = group_by_service(counter_entries)
        for six, (service_name, service_entries) in enumerate(service_groups.items()):
            container_groups = group_by_container(service_entries)
            for ccix, (container_name, container_entries) in enumerate(
                container_groups.items(), 1
            ):
                entries = container_entries

                if len(container_groups) == 1:
                    name = service_name
                else:
                    name = f"{service_name} ({container_name})"

                color = color_map.get(six, {}).get(ccix)
                if color is None:
                    color = colors[color_idx]
                    color_map.setdefault(six, {})[ccix] = color
                    color_idx = (color_idx + 1) % len(colors)

                # Sort the entries by timestamp.
                entries.sort(key=lambda e: e.timestamp)

                fig.add_scatter(
                    x=[e.timestamp - min_timestamp for e in entries],
                    y=[e.counter_value for e in entries],
                    name=name,
                    row=cix,
                    col=1,
                    legendgroup=f"{container_name}-{service_name}",
                    showlegend=cix == 1,
                    line_color=color,
                )

    # Display the full series name in the hover box.
    fig.update_layout(hoverlabel={"namelength": 50})
    # Use SI units for the y-axis.
    fig.update_yaxes(tickformat="s")

    # Show the figure.
    if output_path is None:
        fig.show()
    else:
        fig.write_html(output_path)


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(
        description="Summarizes the performance reports collected from the "
        "containers and presents the data on charts."
    )
    parser.add_argument(
        "reports",
        metavar="REPORT",
        type=str,
        nargs="+",
        help="a path to a performance report",
    )
    parser.add_argument(
        "-o",
        "--output",
        metavar="OUTPUT",
        type=str,
        help="a path to the output file",
        default=None,
    )
    args = parser.parse_args()

    plot_reports(args.reports, args.output)
