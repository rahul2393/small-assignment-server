# noinspection SqlNoDataSourceInspectionForFile
CREATE TABLE `users` (
  `ID` bigint(20) NOT NULL AUTO_INCREMENT,
  `Created` bigint(20) DEFAULT NULL,
  `Updated` bigint(20) DEFAULT NULL,
  `Deleted` tinyint(1) DEFAULT 0,
  `Email` VARCHAR(255) NOT NULL,
  `Name` varchar(255) DEFAULT NULL,
  `PasswordHash` VARCHAR(255) NOT NULL,
  `GroupID`  bigint(20) DEFAULT 3,
  `ExpectedCaloriesPerDay` bigint(20) DEFAULT 0,
  UNIQUE (`Email`),
  PRIMARY KEY (`ID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `tokens` (
  `ID` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `Created` BIGINT(20) NOT NULL,
  `Updated` BIGINT(20) NOT NULL,
  `Deleted` TINYINT(1) NOT NULL,
  `UserID` BIGINT(20) NOT NULL,
  `Hash` VARCHAR(255) NOT NULL,
  `Expiration` BIGINT(20) NOT NULL,
  PRIMARY KEY (`ID`),
  INDEX `UserID` (`UserID` ASC),
  INDEX `deleted` (`Deleted` ASC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `meals` (
  `ID` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `Created` BIGINT(20) NOT NULL,
  `Updated` BIGINT(20) NOT NULL,
  `Deleted` TINYINT(1) NOT NULL,
  `UserID` BIGINT(20) NOT NULL,
  `Description` varchar(255) NOT NULL,
  `MealTime` BIGINT(20) NOT NULL,
  `MealDate` BIGINT(20) NOT NULL,
  `Calories` BIGINT(20) NOT NULL DEFAULT 0,
  PRIMARY KEY (`ID`),
  INDEX `UserID` (`UserID` ASC),
  INDEX `Created` (`Created` ASC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
