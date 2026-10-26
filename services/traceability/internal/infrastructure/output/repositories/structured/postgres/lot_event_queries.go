package postgres

const (
	lotEventSequenceConstraint = "lot_event_lot_sequence_uk"

	selectLotEvent = `
	SELECT id, lot_id, sequence, type, stage, event_date, note, actor_id,
	       actor_role, details, previous_hash, hash, registered_at
	FROM lot_event`

	selectLastLotEvent = selectLotEvent + `
	WHERE lot_id = $1
	ORDER BY sequence DESC
	LIMIT 1`

	selectLotEventsByLot = selectLotEvent + `
	WHERE lot_id = $1
	ORDER BY sequence ASC`

	insertLotEvent = `
		INSERT INTO lot_event (
			id, lot_id, sequence, type, stage, event_date, note, actor_id,
			actor_role, details, previous_hash, hash, registered_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
)
