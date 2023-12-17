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

commit transaction;