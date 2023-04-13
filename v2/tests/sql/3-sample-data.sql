# --- !Up
insert into samples (name) values ('abc');
insert into samples (email) values ('zzz');

# --- !Down
delete from samples where email is not null;
delete from samples where name = 'abc';
