-- name: GetUserName :one
SELECT name FROM users WHERE id=$1;