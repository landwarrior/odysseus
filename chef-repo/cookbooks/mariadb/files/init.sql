CREATE USER IF NOT EXISTS odysseus@localhost identified BY 'odysseus';
CREATE USER IF NOT EXISTS odysseus@'%' identified BY 'odysseus';
GRANT ALL PRIVILEGES ON odysseus.* TO odysseus@localhost;
GRANT ALL PRIVILEGES ON odysseus.* TO odysseus@'%';
