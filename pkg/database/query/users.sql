-- name: GetUser :one
select email, password from users where username=$1 or email=$2;

-- name: NewUser :one
insert into users (username, email, password) values ($1, $2, $3) returning email;
