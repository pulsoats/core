package identity

import "github.com/google/uuid"

func MakeFingerprint(data string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(data))
}
