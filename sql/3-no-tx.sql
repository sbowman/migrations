# --- !Up /notx
insert into samples (name) values ('abc');
insert into samples (blah) values ('zzz'); -- this should fail, but leave abc in the table

# --- !Down /notx
delete from samples where name = 'abc';
delete from samples where blah is not null; -- this should fail, but still delete abc
