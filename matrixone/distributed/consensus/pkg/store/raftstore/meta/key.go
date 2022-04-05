package meta

import "encoding/binary"

const (
	metaKindLogEntry   = 0x01
	metaKindLocalState = 0x02
	metaKindApplyState = 0x03
	metaKindConfState  = 0x04
)

func buildMetaKey(kind byte) []byte {
	key := make([]byte, 1)
	key[0] = kind
	return key
}

func RaftLogEntryKey(index uint64) []byte {
	prefix := buildMetaKey(metaKindLogEntry)
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, index)
	return append(prefix, buf...)
}

func RaftLocalStateKey() []byte {
	return buildMetaKey(metaKindLocalState)
}

func RaftApplyStateKey() []byte {
	return buildMetaKey(metaKindApplyState)
}

func RaftConfStateKey() []byte {
	return buildMetaKey(metaKindConfState)
}
