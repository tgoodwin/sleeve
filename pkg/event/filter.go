package event

func FilterReadsWrites(events []Event) (reads, writes []Event) {
	for _, e := range events {
		if e.OpType == "GET" || e.OpType == "LIST" {
			reads = append(reads, e)
		} else {
			writes = append(writes, e)
		}
	}
	return
}

func IsReadOp(e Event) bool {
	return e.OpType == "GET" || e.OpType == "LIST"
}

func IsWriteOp(e Event) bool {
	return !IsReadOp(e)
}
