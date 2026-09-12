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
new_state AS (
    SELECT 
        s.booking_code_id,
        s.total_legs,
        s.won_legs,
        s.lost_legs,
        s.pending_legs,
        s.void_legs,
        CASE 
            WHEN s.lost_legs > 0 THEN 'LOST'
            WHEN s.pending_legs = 0 AND s.lost_legs = 0 THEN 'WON'
            ELSE 'PENDING'
        END as calculated_status
    FROM stats s
),
updated_tickets AS (
    UPDATE booking_codes bc
    SET status = ns.calculated_status,
        notified_status = CASE 
            WHEN ns.calculated_status IN ('WON', 'LOST') THEN ns.calculated_status 
            ELSE bc.notified_status 
        END,
        updated_at = NOW()
    FROM new_state ns
    WHERE bc.id = ns.booking_code_id
      AND (
          (ns.calculated_status IN ('WON', 'LOST') AND bc.notified_status != ns.calculated_status)
          OR (ns.calculated_status = 'PENDING' AND bc.status != 'PENDING')
      )
    RETURNING bc.id, bc.status, bc.code
)
SELECT 
    ut.id as booking_code_id,
    tkt.id as user_ticket_id,
    ut.code as booking_code,
    ut.status as ticket_status,
    ns.total_legs::int as total_legs,
    ns.won_legs,
    ns.lost_legs,
    ns.pending_legs,
    ns.void_legs,
    tkt.user_id,
    ud.fcm_token
FROM new_state ns
JOIN updated_tickets ut ON ut.id = ns.booking_code_id
JOIN user_tickets tkt ON tkt.booking_code_id = ns.booking_code_id
LEFT JOIN user_devices ud ON ud.user_id = tkt.user_id;

