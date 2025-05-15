BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS "jt808_authcodes" (
	"trackerId"	TEXT NOT NULL UNIQUE,
	"code"	TEXT NOT NULL UNIQUE,
	PRIMARY KEY("trackerId"),
	CONSTRAINT "jt808_authcodes_trackerId_trackers_id" FOREIGN KEY("trackerId") REFERENCES "trackers"("id") ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS "location_data" (
	"id"	INTEGER NOT NULL UNIQUE,
	"trackerId"	TEXT NOT NULL,
	"timestamp"	INTEGER NOT NULL,
	"lat"	INTEGER NOT NULL,
	"lon"	INTEGER NOT NULL,
	"speed"	INTEGER NOT NULL,
	"heading"	INTEGER NOT NULL,
	PRIMARY KEY("id" AUTOINCREMENT),
	CONSTRAINT "fk_location_data_trackerId_trackers_id" FOREIGN KEY("trackerId") REFERENCES "trackers"("id") ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS "models" (
	"name"	TEXT NOT NULL UNIQUE,
	"init_commands"	TEXT NOT NULL,
	"success_keywords"	TEXT NOT NULL,
	PRIMARY KEY("name")
);
CREATE TABLE IF NOT EXISTS "trackers" (
	"id"	TEXT NOT NULL UNIQUE,
	"name"	TEXT NOT NULL UNIQUE,
	"owner"	INTEGER NOT NULL,
	"phoneNumber"	TEXT NOT NULL,
	"model"	TEXT NOT NULL,
	"enabled"	INTEGER NOT NULL,
	"lastConnected"	INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY("id"),
	CONSTRAINT "fk_trackers_model__models_name" FOREIGN KEY("model") REFERENCES "models"("name"),
	CONSTRAINT "fk_trackers_owner__users_id" FOREIGN KEY("owner") REFERENCES "users"("id")
);
CREATE TABLE IF NOT EXISTS "users" (
	"id"	INTEGER NOT NULL UNIQUE,
	"name"	TEXT NOT NULL UNIQUE,
	"apikey"	TEXT NOT NULL UNIQUE,
	"admin"	INTEGER NOT NULL,
	"enabled"	INTEGER NOT NULL,
	PRIMARY KEY("id" AUTOINCREMENT)
);
INSERT INTO "models" ("name","init_commands","success_keywords") VALUES ('W18L','SERVER,1,<ip>,<port>,0#;GMT,E,0,0#;HBT,5#','OK!;OK!;OK!'),
 ('R56','SERVER,8520,<ip>,<port>,0#;GMT,E,0,0#;HBT,5#;SLPON#;DEEPSLP,1#','OK!;OK!;OK!;ON!;ON!'),
 ('D21L','SERVER,0,<ip>,<port>,0#;GMT,E,0,0#;HBT,5#','OK!;OK!;OK!'),
 ('R58L','<HL&P:HOLLOO&B:<ip>:<port>&1H:300,3600>','<ip>:<port>&1H:300,3600');
INSERT INTO "users" ("name","apikey","admin","enabled") VALUES ('Admin','AAAAAA',1,1);
COMMIT;


