update gophermart."order"
set
    accrual_readiness = false,
    accrual_started_at = now()
where id in
      (select id
      from gophermart."order"
      where status not in ('INVALID', 'PROCESSED') and accrual_readiness = true
      order by uploaded_at desc
      limit 10)
returning
    id, number, user_login, uploaded_at, coalesce(accrual, 0), status;

