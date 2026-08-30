-- name: EvaluateTickets :many
WITH stats AS (
    SELECT 
        booking_code_id,
        COUNT(*) as total_legs,
        COUNT(*) FILTER (WHERE status = 'WON')::int as won_legs,
        COUNT(*) FILTER (WHERE status = 'LOST')::int as lost_legs,
        COUNT(*) FILTER (WHERE status = 'PENDING')::int as pending_legs,
        COUNT(*) FILTER (WHERE status = 'VOID')::int as void_legs
    FROM booking_selections
    WHERE booking_code_id = ANY(@booking_code_ids::uuid[])
    GROUP BY booking_code_id
),
updated_tickets AS (
    UPDATE booking_codes bc
    SET status = CASE 
        WHEN s.lost_legs > 0 THEN 'LOST'
        WHEN s.pending_legs = 0 AND s.lost_legs = 0 THEN 'WON'
        ELSE 'PENDING'
    END
    FROM stats s
    WHERE bc.id = s.booking_code_id 
    RETURNING bc.id, bc.status, bc.code
)
SELECT 
    ut.id as booking_code_id,
    ut.code as booking_code,
    ut.status as ticket_status,
    s.total_legs::int as total_legs,
    s.won_legs,
    s.lost_legs,
    s.pending_legs,
    s.void_legs,
    tkt.user_id,
    ud.fcm_token
FROM stats s
JOIN updated_tickets ut ON ut.id = s.booking_code_id
JOIN user_tickets tkt ON tkt.booking_code_id = s.booking_code_id
LEFT JOIN user_devices ud ON ud.user_id = tkt.user_id;
