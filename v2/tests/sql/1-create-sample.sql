# --- !Up
create table samples (
    name varchar(64) primary key
);

# --- !Down
drop table samples
