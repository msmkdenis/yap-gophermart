begin transaction;

create schema if not exists gophermart;

create table if not exists gophermart.user
(
    id                      text,
    login                   text unique not null,
    password                bytea unique not null,
    constraint pk_user primary key (id)
);

create type gophermart.order_status as enum
    ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');

create table if not exists gophermart.order
(
    id                      text,
    number                  text unique not null,
    user_login              text not null,
    uploaded_at             timestamp default now() not null,
    status                  gophermart.order_status not null,
    accrual                 numeric(10,2),
    constraint pk_order primary key (id),
    constraint fk_user foreign key (user_login) references gophermart.user (login)
);

create table if not exists gophermart.balance
(
    id                      uuid default gen_random_uuid(),
    user_login              text not null,
    current                 numeric(10,2) default 0 check ( current >= 0),
    withdrawn               numeric(10,2) default 0 check ( withdrawn >= 0),
    constraint pk_balance primary key (id),
    constraint fk_user foreign key (user_login) references gophermart.user (login)
);

create or replace function gophermart.register_create_balance()
    returns trigger
    language plpgsql
as $$
begin
    insert into gophermart.balance (user_login)
    values (new.login);
    return new;
END
$$;

create or replace trigger create_balance
    after insert on gophermart.user
    for each row
execute function gophermart.register_create_balance();

create table if not exists gophermart.withdrawals
(
    id                      uuid default gen_random_uuid(),
    order_number            text not null,
    user_login              text not null,
    sum                     numeric(10,2) not null check ( sum >= 0),
    processed_at            timestamp default now(),
    constraint pk_withdrawals primary key (id),
    constraint fk_user foreign key (user_login) references gophermart.user (login)
);

commit transaction;