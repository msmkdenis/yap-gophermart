select id, login, password
from gophermart.user
where login = $1;