import json
from dataclasses import dataclass
from collections import defaultdict

READ_OPERATIONS = {"GET", "LIST"}
WRITE_OPERATIONS = {"CREATE", "PATCH", "UPDATE", "DELETE"}


@dataclass
class Event:
    timestamp: str
    reconcile_id: str
    controller_id: str
    root_event_id: str
    op_type: str
    kind: str
    object_id: str
    version: str
    labels: dict = None

    @classmethod
    def from_json(self, event_json):
        obj = json.loads(event_json)
        # get every key in the dictionary that starts with the prefix "label:discrete.events"
        labels = {
            k[len("label:discrete.events/") :]: v
            for k, v in obj.items()
            if k.startswith("label:discrete.events/")
        }
        return Event(
            timestamp=obj.get("timestamp"),
            reconcile_id=obj.get("reconcile_id"),
            controller_id=obj.get("controller_id"),
            root_event_id=obj.get("root_event_id"),
            op_type=obj.get("op_type"),
            kind=obj.get("kind"),
            object_id=obj.get("object_id"),
            version=obj.get("version"),
            labels=labels,
        )

    def label(self, key):
        return self.labels.get(key)

    def __repr__(self) -> str:
        obj_id_short = self.object_id.split("-")[0]
        return f"Object({self.kind}, id={obj_id_short}, version={self.version} change-id={self.label('change-id')}, op={self.op_type})"


def backfill_labels(events):
    """
    Events with CREATE op type are missing version, kind, and object_id.
    The 'full' object may exist within a later read event.
    We can correlate the CREATE event with the later read event by the 'change-id' label.
    So, we can backfill the missing fields from the read events.
    """
    read_events = [e for e in events if e.op_type in READ_OPERATIONS]
    by_change_id = {}
    for event in read_events:
        change_id = event.label("change-id")
        if change_id:
            by_change_id[change_id] = event

    create_events = [e for e in events if e.op_type == "CREATE"]
    for create_event in create_events:
        change_id = create_event.label("change-id")
        if change_id not in by_change_id:
            continue
            # raise RuntimeError(f"Missing read event for CREATE event: {create_event}")
        read_event = by_change_id[change_id]
        create_event.kind = read_event.kind
        create_event.object_id = read_event.object_id
        create_event.version = read_event.version

    return events



def process(lines):
    by_reconcile_id = defaultdict(list)
    events = []
    for line in lines:
        # validate the line is a JSON object
        if not line.startswith("{"):
            raise ValueError(f"Invalid JSON object: {line}")

        # parse the JSON object
        event = Event.from_json(line)
        events.append(event)

    backfill_labels(events)
    for event in events:
        reconcile_id = event.reconcile_id
        by_reconcile_id[reconcile_id].append(event)

    # sanity checks
    for reconcile_id, events in by_reconcile_id.items():
        read_events = [e for e in events if e.op_type in READ_OPERATIONS]
        write_events = [e for e in events if e.op_type in WRITE_OPERATIONS]
        assert len(read_events) + len(write_events) == len(events)
        assert len(set([e.controller_id for e in events])) == 1

    reads_to_writes = readsets_to_writesets(by_reconcile_id)
    for reconcile_id, rw in reads_to_writes.items():
        if len(rw["writeset"]) == 0:
            continue

        print(f"Reconcile ID: {reconcile_id}, Controller: {rw['writeset'][0].controller_id}")
        print("Readset:")
        for event in rw["readset"]:
            print(f"\t{event}")
        print("Writeset:")
        for event in rw["writeset"]:
            print(f"\t{event}")


def readsets_to_writesets(events_by_reconcile_id):
    out = {}
    for reconcile_id, events in events_by_reconcile_id.items():
        readset = [e for e in events if e.op_type in READ_OPERATIONS]
        writeset = [e for e in events if e.op_type in WRITE_OPERATIONS]
        assert len(readset) + len(writeset) == len(events)
        out[reconcile_id] = {"readset": readset, "writeset": writeset}

    return out


def main():
    if len(sys.argv) > 1:
        with open(sys.argv[1], "r") as infile:
            lines = infile.readlines()
            process(lines)

    return 0


if __name__ == "__main__":
    import sys

    sys.exit(main())
