create schema if not exists gophermart;

create table if not exists gophermart.user
(
    id                      text,
    login                   text unique not null,
    password                bytea unique not null,
    constraint pk_url primary key (id)
);