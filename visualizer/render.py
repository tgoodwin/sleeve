import json
from typing import Optional
import matplotlib.pyplot as plt
from dataclasses import dataclass
from collections import defaultdict
import uuid
import colorsys


def generate_pastel_colors(num_colors):
    """
    Generates a list of pastel hex colors evenly spaced around the color wheel.

    Parameters:
    num_colors (int): The number of distinct pastel colors to generate.

    Returns:
    list: A list of hex color codes.
    """
    pastel_colors = []

    # Evenly space hues around the color wheel
    for i in range(num_colors):
        hue = i / num_colors  # Evenly spaced hues
        # Use high value and low-to-medium saturation to get pastel-like colors
        rgb = colorsys.hsv_to_rgb(hue, 0.4, 1.0)  # S = 0.4 for pastel, V = 1.0 for brightness
        # Convert to hex
        pastel_colors.append('#{:02x}{:02x}{:02x}'.format(int(rgb[0] * 255), int(rgb[1] * 255), int(rgb[2] * 255)))

    return pastel_colors

def assign_colors_to_ids(ids):
    """
    Assigns a unique pastel color to each ID.

    Parameters:
    ids (list): A list of unique IDs.

    Returns:
    dict: A dictionary mapping each ID to a unique pastel color.
    """
    num_ids = len(ids)
    pastel_colors = generate_pastel_colors(num_ids)
    return {id_: color for id_, color in zip(ids, pastel_colors)}


@dataclass
class Event:
    timestamp: int
    process_id: int
    reconcile_id: int
    sender_id: Optional[str] = None
    receiver_id: Optional[str] = None
    id: Optional[str] = None

    def __post_init__(self):
        if self.id is None:
            self.id = str(uuid.uuid4())
        self._validate_event()

    def _validate_event(self):
        if self.sender_id and self.receiver_id:
            raise ValueError("An event cannot have both sender_id and receiver_id populated.")
        if self.sender_id and self.sender_id == self.process_id:
            raise ValueError("Sender cannot be the same as the process.")
        if self.receiver_id and self.receiver_id == self.process_id:
            raise ValueError("Receiver cannot be the same as the process.")

    def is_send_event(self):
        return self.receiver_id is not None

    def is_receive_event(self):
        return self.sender_id is not None

    def __repr__(self):
        return f"Event({self.timestamp}, {self.process_id}, sender={self.sender_id}, receiver={self.receiver_id})"


class LamportDiagram:
    def __init__(self):
        # Maps process_id to a list of events
        self.processes = defaultdict(list)
        self.events = []

        self.events_by_id = {}

    def add_event(self, event):
        self.processes[event.process_id].append(event)
        self.events.append(event)
        self.events_by_id[event.id] = event

    def render(self):

        fig, ax = plt.subplots()

        # Drawing process lines
        process_y_positions = {}

        events_by_reconcile_id = defaultdict(list)

        event_x_positions = {event.id: event.timestamp for event in self.events}
        max_x = max(event_x_positions.values()) + 1
        for i, process_id in enumerate(sorted(self.processes.keys())):
            process_y_positions[process_id] = i
            ax.plot([0, max_x], [i, i], 'k-')  # Draw a horizontal line for each process
            ax.text(-0.5, i + 0.1, f"Process {process_id}", verticalalignment='center')

        # Drawing events
        for event in sorted(self.events, key=lambda e: e.timestamp):
            x = event_x_positions[event.id]
            y = process_y_positions[event.process_id]
            events_by_reconcile_id[event.reconcile_id].append(event)
            ax.plot(x, y, 'bo', zorder=1)  # Plot event as a dot
            ax.text(x + 0.1, y + 0.1, f"Event {event.timestamp}", verticalalignment='center')

            # Drawing arrows for communication
            # TODO handle case when GET event is on the same process but is not the most recent value
            if event.is_send_event():
                receiver_y = process_y_positions[self.events_by_id[event.receiver_id].process_id]
                receiver_x = event_x_positions[event.receiver_id]
                ax.arrow(x, y, receiver_x - x, receiver_y - y, head_width=0.1, head_length=0.1, fc='r', ec='r', zorder=2)
            elif event.is_receive_event():
                sender_y = process_y_positions[self.events_by_id[event.sender_id].process_id]
                sender_x = event_x_positions[event.sender_id]
                ax.arrow(sender_x, sender_y, x - sender_x, y - sender_y, head_width=0.1, head_length=0.1, fc='r', ec='r', zorder=2)

        reconcile_id_color = assign_colors_to_ids(set(event.reconcile_id for event in self.events))
        for reconcile_id, events in events_by_reconcile_id.items():
            process_id = events[0].process_id
            y_pos = process_y_positions[process_id]
            positions = [event_x_positions[e.id] for e in events]
            min_x = min(positions)
            max_x = max(positions)
            # reconcile_id_color[reconcile_id] = get_a_color(reconcile_id)
            plt.gca().add_patch(
                plt.Rectangle(
                    (min_x - 0.2, y_pos - 0.2),
                    max_x - min_x + 0.4, 0.4,
                    color=reconcile_id_color[reconcile_id],
                    edgecolor='g',
                    zorder=0)
                )



        # Create legend for reconcile_id colors
        legend_elements = []
        for reconcile_id, color in reconcile_id_color.items():
            legend_elements.append(plt.Line2D([0], [0], marker='o', color='w', label=f"Reconcile {reconcile_id}", markerfacecolor=color, markersize=10))
        ax.legend(handles=legend_elements, loc='upper right')

        # Set axis limits
        ax.set_xlim(-1, max_x)
        ax.set_ylim(-1, len(self.processes))
        ax.set_yticks([])

        plt.show()


def convert_data_to_events(data):
    change_id_map = {}
    events = []

    events_by_id = {}
    senders_by_change_id = {}

    # First pass: Collect all send events and map them by `change-id`
    for row in data:
        uid = str(uuid.uuid4())
        events_by_id[uid] = row
        row["uid"] = uid
        if row["OpType"] in {"CREATE", "UPDATE", "DELETE"}:
            change_id = row.get("label:change-id")
            if change_id:
                change_id_map[change_id] = row
                senders_by_change_id[change_id] = row["uid"]

    # Second pass: Create events and link GET events to corresponding send events
    for row in data:
        timestamp = row["Timestamp"]
        process_id = row["Controller"]
        op_type = row["OpType"]
        reconcile_id = row["ReconcileID"]

        uid = row["uid"]

        if op_type in {"CREATE", "UPDATE", "DELETE"}:
            receiver_id = None  # Send event
            event = Event(timestamp, process_id, reconcile_id, id=uid)
            events.append(event)

        elif op_type == "GET":
            change_id = row.get("label:change-id")
            if change_id and change_id in senders_by_change_id:
                sender_id = senders_by_change_id[change_id]
                # sender_id = change_id_map[change_id]["Controller"]
                event = Event(timestamp, process_id, reconcile_id, sender_id=sender_id, id=uid)
                events.append(event)
            else:
                # Handle cases where there is no matching send event
                event = Event(timestamp, process_id, reconcile_id)  # Isolated GET event, no corresponding send event found
                events.append(event)
        else:
            raise ValueError(f"Unknown op_type: {op_type}")

    return events

def parse(file):
    with open(file, 'r') as f:
        data = f.read().splitlines()
        return [json.loads(row) for row in data]


# Convert the data to Event instances
def main(data):
    events = convert_data_to_events(data)
    lamport = LamportDiagram()
    for event in events:
        print(event)
        lamport.add_event(event)

    lamport.render()

if __name__ == "__main__":
    import sys
    if len(sys.argv) < 2:
        print("Usage: python render.py <input-file>")
        sys.exit(1)

    # newline-delimited JSON file
    infile = sys.argv[1]
    data = parse(infile)
    sys.exit(main(data))