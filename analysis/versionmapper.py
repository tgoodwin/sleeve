import json
from dataclasses import dataclass

@dataclass
class Version:
    kind: str
    object_id: str
    version: str
    value: object

    @classmethod
    def from_json(cls, version_json):
        obj = json.loads(version_json)
        value_json = obj.get("value")
        value = json.loads(value_json)

        return Version(
            # TODO hacky
            kind=obj.get("kind").split("Kind=")[1],
            object_id=obj.get("object_id"),
            version=obj.get("version"),
            value=value
        )

    def id(self):
        return f'{self.kind}/{self.causal_id()}'

    def causal_id(self):
        labels = self.labels()
        change_id_label = labels.get("discrete.events/change-id")
        if change_id_label:
            return change_id_label
        if not change_id_label:
            root_label = labels.get("tracey-uid")
            if not root_label:
                raise ValueError(f"Missing change-id and root-event-id labels: {self.value}")
            return root_label

    def labels(self):
        return self.value.get("metadata", {}).get("labels", None)

    def annotations(self):
        metadata = self.value.get("metadata", {})
        return metadata.get("annotations", None)

    def status_conditions(self):
        annotations = self.annotations()
        if annotations:
            return {k: v for k, v in annotations.items() if k.startswith("status.")}
        return None

def process(lines):
    versions = {}
    for line in lines:
        v = Version.from_json(line)
        print(v.id())
        versions[v.id()] = v
        # annotations = v.status_conditions()
        # # labels = v.labels()
        # causal_label = v.causal_id()
        # print(causal_label, annotations)
    return versions

def main():
    if len(sys.argv) > 1:
        with open(sys.argv[1], "r") as infile:
            lines = infile.readlines()
            process(lines)

if __name__ == '__main__':
    import sys

    sys.exit(main())