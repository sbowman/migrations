--- !Up
create table user_roles
(
    user_id int not null,
    role_id int not null,
    primary key (user_id, role_id)
);

--- !Down
drop table user_roles;
