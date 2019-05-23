# --- !Up
create table sample (
    name varchar(64) primary key
);

# --- !Down
drop table sample
