--- !Up
create table roles
(
    id   serial primary key,
    name varchar(32) not null
);

create unique index idx_role_name on roles (name);

--- !Down
drop table roles;
