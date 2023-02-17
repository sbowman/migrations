# --- !Up
insert into samples (name) values ('abc');
insert into samples (blah) values ('zzz');

# --- !Down
delete from samples where name = 'abc';
delete from samples where blah is not null;
