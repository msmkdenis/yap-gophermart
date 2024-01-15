update gophermart.order
set
    accrual = $1,
    status = $2,
    accrual_readiness =
        case
            when $2::gophermart.order_status in ('PROCESSED', 'INVALID') then false
            else true
        end,
    accrual_finished_at = now(),
    accrual_count = accrual_count + 1
where number = $3;