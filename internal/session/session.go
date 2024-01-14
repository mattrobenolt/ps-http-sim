package session

import (
	"math/rand"

	gonanoid "github.com/matoous/go-nanoid/v2"
	psdbv1alpha1 "github.com/planetscale/psdb/types/psdb/v1alpha1"
	querypb "github.com/planetscale/vitess-types/gen/vitess/query/v16"
	vtgatepb "github.com/planetscale/vitess-types/gen/vitess/vtgate/v16"
	"vitess.io/vitess/go/sqltypes"
)

func UUID(session *psdbv1alpha1.Session) string {
	if session != nil && session.VitessSession != nil {
		return session.VitessSession.SessionUUID
	}
	return ""
}

func DBName(session *psdbv1alpha1.Session) string {
	if session != nil && session.VitessSession != nil {
		return session.VitessSession.TargetString
	}
	return ""
}

func Update(qr *sqltypes.Result, session *psdbv1alpha1.Session) {
	if s := session.VitessSession; qr != nil && s != nil {
		s.LastInsertId = qr.InsertID
		s.InTransaction = qr.IsInTransaction()
		s.FoundRows = uint64(len(qr.Rows))
		s.RowCount = int64(qr.RowsAffected)
	}
}

func Reset(session *psdbv1alpha1.Session) *psdbv1alpha1.Session {
	id := UUID(session)
	dbname := DBName(session)

	session = New(dbname)
	session.VitessSession.SessionUUID = id
	return session
}

func New(dbname string) *psdbv1alpha1.Session {
	// we're not doing anything with the signature, and it's opaque bytes
	// to clients, so just generate a random 32 bytes
	var signature [32]byte
	rand.Read(signature[:])

	session := &psdbv1alpha1.Session{
		Signature: signature[:],
		VitessSession: &vtgatepb.Session{
			TargetString: dbname,
			Options: &querypb.ExecuteOptions{
				IncludedFields:  querypb.ExecuteOptions_ALL,
				Workload:        querypb.ExecuteOptions_UNSPECIFIED,
				ClientFoundRows: true,
			},
			Autocommit:           true,
			DDLStrategy:          "direct",
			SessionUUID:          gonanoid.Must(),
			EnableSystemSettings: true,
		},
	}
	return session
}
