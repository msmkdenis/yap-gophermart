update gophermart.balance
set current = current + $1
where user_login = $2;