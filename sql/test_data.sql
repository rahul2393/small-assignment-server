Delete from users;
Alter table users auto_increment = 0;
INSERT INTO users VALUES (1, 1435350391835, 1435350391835, 0, "rahul.agrawal@hotcocoasoftware.com", "Rahul Agrawal", "$2a$10$E4GaCw/xw2O.T6dhGD2So.PT.hXzLDoLn.NeRFhg.Sy30Q6xd5VXa", 1, 1200);
INSERT INTO users VALUES (2, 1435350391835, 1435350391835, 0, "rahul.yadav@hotcocoasoftware.com", "Rahul Yadav", "$2a$10$E4GaCw/xw2O.T6dhGD2So.PT.hXzLDoLn.NeRFhg.Sy30Q6xd5VXa", 2, 1500);
INSERT INTO users VALUES (3, 1435350391835, 1435350391835, 0, "ritik.rishu@hotcocoasoftware.com", "Ritik Rishu", "$2a$10$E4GaCw/xw2O.T6dhGD2So.PT.hXzLDoLn.NeRFhg.Sy30Q6xd5VXa", 3, 2000);

Delete from tokens;
Delete from meals;
