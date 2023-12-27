update gophermart.balance
set current = current - $1, withdrawn = withdrawn + $1
where user_login = $2;