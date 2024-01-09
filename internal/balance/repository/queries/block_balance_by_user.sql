select
    id, user_login, current, withdrawn
from gophermart.balance
where user_login = $1
for update;