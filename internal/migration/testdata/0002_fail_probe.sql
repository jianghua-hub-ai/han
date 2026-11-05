CREATE TABLE rolled_back_probe (
    id integer PRIMARY KEY
);

INSERT INTO table_that_does_not_exist (id) VALUES (2);
