-- name: GetUserStats :one
SELECT 
    total_tickets, won_tickets, lost_tickets, pending_tickets,
    total_staked, total_returns,
    (total_returns - total_staked)::numeric AS net_profit
FROM user_stats
WHERE user_id = $1;
