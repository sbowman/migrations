--- !Up
alter table samples add column email varchar(1024);
create unique index idx_sample_email on samples (email);

--- !Down
alter table samples drop column email;
