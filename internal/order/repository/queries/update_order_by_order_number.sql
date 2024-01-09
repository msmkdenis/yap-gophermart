update gophermart.order
set accrual = $1, status = $2
where number = $3;