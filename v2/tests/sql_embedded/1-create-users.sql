# --- !Up
create table users
(
    id       serial primary key,
    email    varchar(256) not null,
    username varchar(64)  not null
);

create unique index idx_user_email on users (email);
create unique index idx_user_username on users (username);

# --- !Down
drop table users;
