-- DROP DATABASE IF EXISTS registry ;
-- CREATE DATABASE registry ;
-- USE registry ;
-- ^^ OLD STUF

-- MySQL config requirements:
-- sql_mode:
--   ANSI_QUOTES        <- enabled
--   ONLY_FULL_GROUP_BY <- disabled

/*
Notes:
SID -> System generated ID, usually primary key. Assumed to be globally
unique so that when searching we don't need to combine it with any other
scoping column to provide uniqueness
UID -> User provided ID. Only needs to be unique within its parent scope.
This is why we use SID for cross-table joins/links.
The code doesn't do delete propagation. Instead, the code will delete
whichever resource was asked to be deleted and then the DB triggers
will delete all necessarily (related) rows/resources as needed. So,
deleting a row from the "Registry" table should delete ALL other resources
in all other tables automatically.
The "Props" table holds all properties for all entities rather than
having property specific columns in the appropriate tables. No idea which
is easier/faster but having it all in one table made things a lot easier
for filtering/searching. But we can switch it if needed at some point. This
also means that all properties (including extensions) are processed the
same way... via the generic Get/Set methods.
*/


SET GLOBAL sql_mode = 'ANSI_QUOTES' ;
SET sql_mode = 'ANSI_QUOTES' ;

CREATE TABLE Registries (
    SID     VARCHAR(255) NOT NULL,  # System ID
    UID     VARCHAR(255) NOT NULL,  # User defined

    PRIMARY KEY (SID),
    UNIQUE INDEX (UID)
);

CREATE TRIGGER RegistryTrigger BEFORE DELETE ON Registries
FOR EACH ROW
BEGIN
    DELETE FROM Props    WHERE RegistrySID=OLD.SID $$
    DELETE FROM "Groups" WHERE RegistrySID=OLD.SID $$
    DELETE FROM Models   WHERE RegistrySID=OLD.SID $$
END ;

CREATE TABLE Models (
    RegistrySID VARCHAR(64) NOT NULL,

    Model       JSON,               # Full model, not just Registry
    Labels      JSON,
    Attributes  JSON,               # Until we use the Attributes table

    PRIMARY KEY (RegistrySID)
);

CREATE TRIGGER ModelsTrigger BEFORE DELETE ON Models
FOR EACH ROW
BEGIN
    DELETE FROM ModelEntities WHERE RegistrySID=OLD.RegistrySID $$
END ;

CREATE TABLE ModelEntities (        # Group or Resource (no parent=Group)
    SID               VARCHAR(255),       # my System ID
    RegistrySID       VARCHAR(64),
    ParentSID         VARCHAR(64),        # ID of parent ModelEntity
    Abstract          VARCHAR(255),       # /GROUPS, /GROUPS/RESOURCES

    # For Groups and Resources
    Plural            VARCHAR(64),
    Singular          VARCHAR(64),
    Description       VARCHAR(255),
    ModelVersion      VARCHAR(255),
    CompatibleWith    VARCHAR(255),
    Labels            JSON,
    XImportResources  VARCHAR($MAX_VARCHAR),
    Attributes        JSON,               # Until we use the Attributes table

    # For Resources
    MaxVersions       INT,
    SetVersionId      BOOL,
    SetDefaultSticky  BOOL,
    HasDocument       BOOL,
    SingleVersionRoot BOOL,
    TypeMap           JSON,
    MetaAttributes    JSON,

    PRIMARY KEY(SID),
    UNIQUE INDEX (RegistrySID, ParentSID, Plural),
    UNIQUE INDEX (RegistrySID, Abstract),
    CONSTRAINT UC_Singular UNIQUE (RegistrySID, ParentSID, Singular)
);

CREATE TRIGGER ModelTrigger BEFORE DELETE ON ModelEntities
FOR EACH ROW
BEGIN
    DELETE FROM "Groups"        WHERE ModelSID=OLD.SID $$
    DELETE FROM Resources       WHERE ModelSID=OLD.SID $$
    DELETE FROM ModelAttributes WHERE ParentSID=OLD.SID $$
END ;

# Not used yet
CREATE TABLE ModelAttributes (
    SID           VARCHAR(64) NOT NULL,   # my System ID
    RegistrySID   VARCHAR(64) NOT NULL,
    ParentSID     VARCHAR(64),            # NULL=Root. Model or IfValue SID
    Name          VARCHAR(64) NOT NULL,
    Type          VARCHAR(64) NOT NULL,
    Description   VARCHAR(255),
    Strict        BOOL NOT NULL,
    Required      BOOL NOT NULL,
    ItemType      VARCHAR(64),

    PRIMARY KEY(RegistrySID, ParentSID, SID),
    UNIQUE INDEX (SID),
    CONSTRAINT UC_Name UNIQUE (RegistrySID, ParentSID, Name)
);

CREATE TRIGGER ModelAttributeTrigger BEFORE DELETE ON ModelAttributes
FOR EACH ROW
BEGIN
    DELETE FROM ModelEnums    WHERE AttributeSID=OLD.SID $$
    DELETE FROM ModelIfValues WHERE AttributeSID=OLD.SID $$
END ;

CREATE TABLE ModelEnums (
    RegistrySID   VARCHAR(64) NOT NULL,
    AttributeSID  VARCHAR(64) NOT NULL,
    Value         VARCHAR(255) NOT NULL,

    PRIMARY KEY(RegistrySID, AttributeSID),
    INDEX (AttributeSID),
    CONSTRAINT UC_Value UNIQUE (RegistrySID, AttributeSID, Value)
);

CREATE TABLE ModelIfValues (
    SID           VARCHAR(64) NOT NULL,
    RegistrySID   VARCHAR(64) NOT NULL,
    AttributeSID  VARCHAR(64) NOT NULL,
    Value         VARCHAR(255) NOT NULL,

    PRIMARY KEY(RegistrySID, AttributeSID),
    UNIQUE INDEX (SID),
    INDEX (AttributeSID),
    CONSTRAINT UC_Value UNIQUE (RegistrySID, AttributeSID, Value)
);

CREATE TRIGGER ModelIfValuesTrigger BEFORE DELETE ON ModelIfValues
FOR EACH ROW
BEGIN
    DELETE FROM ModelAttributes    WHERE ParentSID=OLD.SID $$
END ;


CREATE TABLE "Groups" (
    SID             VARCHAR(64) NOT NULL,   # System ID
    UID             VARCHAR(64) NOT NULL,   # User defined
    RegistrySID     VARCHAR(64) NOT NULL,
    ModelSID        VARCHAR(64) NOT NULL,
    Path            VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Abstract        VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Plural          VARCHAR(64) NOT NULL,
    Singular        VARCHAR(64) NOT NULL,

    PRIMARY KEY (SID),
    INDEX(RegistrySID, UID),
    UNIQUE INDEX (RegistrySID, ModelSID, UID)
);

CREATE TRIGGER GroupTrigger BEFORE DELETE ON "Groups"
FOR EACH ROW
BEGIN
    DELETE FROM Props WHERE EntitySID=OLD.SID $$
    DELETE FROM Resources WHERE GroupSID=OLD.SID $$
END ;

CREATE TABLE Resources (
    SID             VARCHAR(64) NOT NULL,   # System ID
    UID             VARCHAR(64) NOT NULL,   # User defined
    RegistrySID     VARCHAR(64) NOT NULL,
    GroupSID        VARCHAR(64) NOT NULL,   # System ID
    ModelSID        VARCHAR(64) NOT NULL,
    Path            VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Abstract        VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Plural          VARCHAR(64) NOT NULL,
    Singular        VARCHAR(64) NOT NULL,

    PRIMARY KEY (SID),
    UNIQUE INDEX(RegistrySID,SID),
    INDEX(GroupSID, UID),
    INDEX(Path),
    INDEX(RegistrySID),
    UNIQUE INDEX (GroupSID, ModelSID, UID)
);

CREATE TRIGGER ResourcesTrigger BEFORE DELETE ON Resources
FOR EACH ROW
BEGIN
    DELETE FROM Props WHERE EntitySID=OLD.SID $$
    DELETE FROM Metas WHERE ResourceSID=OLD.SID $$
    DELETE FROM Versions WHERE ResourceSID=OLD.SID $$
END ;

CREATE TABLE Metas (
    SID             VARCHAR(64) NOT NULL,   # System ID
    RegistrySID     VARCHAR(64) NOT NULL,
    ResourceSID     VARCHAR(64) NOT NULL,   # System ID
    Path            VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Abstract        VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Plural          VARCHAR(64) NOT NULL,
    Singular        VARCHAR(64) NOT NULL,

    xRefSID         VARCHAR(64),           # Generated
    defaultVID      VARCHAR(64),           # Generated

    PRIMARY KEY (SID),
    UNIQUE INDEX(RegistrySID,SID),
    INDEX(RegistrySID, ResourceSID),
    INDEX(RegistrySID, Path),
    INDEX(RegistrySID),
    INDEX(RegistrySID,xRefSID)
);

# Can't use this because we get recursive triggers on meta.delete()
# CREATE TRIGGER MetasTrigger BEFORE DELETE ON Metas
# FOR EACH ROW
# BEGIN
    # DELETE FROM Props WHERE EntitySID=OLD.SID $$
# END ;

CREATE TABLE Versions (
    SID                 VARCHAR(64) NOT NULL,   # System ID
    UID                 VARCHAR(64) NOT NULL,   # User defined
    RegistrySID         VARCHAR(64) NOT NULL,
    ResourceSID         VARCHAR(64) NOT NULL,   # System ID
    Path                VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    Abstract            VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,

    Ancestor            VARCHAR(65) NOT NULL COLLATE utf8mb4_bin,  # Generated
    CreatedAt           VARCHAR(255),           # Generated (for ancestor stuff

    PRIMARY KEY (SID),
    UNIQUE INDEX (ResourceSID, UID),
    UNIQUE INDEX (RegistrySID, SID),
    INDEX (ResourceSID),
    INDEX (RegistrySID, ResourceSID, Ancestor)
);

CREATE TABLE Props (
    RegistrySID VARCHAR(64) NOT NULL,
    EntitySID   VARCHAR(64) NOT NULL,       # Reg,Group,Res,Ver System ID
    eType       INT NOT NULL,
    PropName    VARCHAR($MAX_PROPNAME) NOT NULL,
    PropValue   VARCHAR($MAX_VARCHAR),
    PropType    CHAR(64) NOT NULL,          # string, boolean, int, ...
    DocView     BOOL NOT NULL,              # Should include during doc view?

    # non-doc-view-able attributes are ones that are generated at runtime
    # due to things like showing the Default Version props in the Resource
    # or entities/props that materialize due to an xref. Normally a GET
    # will show all props, but during /export or ?doc we want to exclude
    # these non-doc-view ones. In case where all of the props for an entity
    # are generated, the entire entity should vanish from the serialization.
    # e.g. Versions of an xref'd Resource.

    PRIMARY KEY (EntitySID, PropName),
    INDEX (EntitySID),
    INDEX (RegistrySID, PropName)
);

CREATE TRIGGER PropsAncestor BEFORE INSERT ON Props
FOR EACH ROW
BEGIN
    IF (NEW.eType=$ENTITY_VERSION) THEN
        IF (NEW.PropName='ancestor$DB_IN') THEN
          UPDATE Versions SET Ancestor=NEW.PropValue
              WHERE SID=NEW.EntitySID $$
        END IF $$
        IF (NEW.PropName='createdat$DB_IN') THEN
          UPDATE Versions SET CreatedAt=NEW.PropValue
              WHERE SID=NEW.EntitySID $$
        END IF $$
    END IF $$

    IF (NEW.eType=$ENTITY_META) THEN
        IF (NEW.PropName='xref$DB_IN') THEN
          # Remove leading /
          SET @rSID := (SELECT SID FROM Resources WHERE
                        RegistrySID=NEW.RegistrySID AND
                        Path=SUBSTRING(NEW.PropValue,2)) $$

          UPDATE Metas AS m SET xRefSID=@rSID
            WHERE m.SID=NEW.EntitySID $$
        END IF $$
        IF (NEW.PropName='defaultversionid$DB_IN') THEN
          UPDATE Metas AS m SET defaultVID=NEW.PropValue
            WHERE m.SID=NEW.EntitySID $$
        END IF $$
    END IF $$
END ;

CREATE TRIGGER PropsXref BEFORE DELETE ON Props
FOR EACH ROW
BEGIN
    IF (OLD.eType=$ENTITY_META) THEN
        IF (OLD.PropName='xref$DB_IN') THEN
          UPDATE Metas SET xRefSID=NULL
          WHERE SID=OLD.EntitySID $$
        END IF $$
        IF (OLD.PropName='defaultversionid$DB_IN') THEN
          UPDATE Metas AS m SET defaultVID=NULL
            WHERE m.SID=OLD.EntitySID $$
        END IF $$
    END IF $$
END ;

CREATE TRIGGER VersionsTrigger BEFORE DELETE ON Versions
FOR EACH ROW
BEGIN
    DELETE FROM Props WHERE EntitySID=OLD.SID $$
    DELETE FROM ResourceContents WHERE VersionSID=OLD.SID $$
END ;

CREATE VIEW Entities AS
SELECT                          # Gather Registries
    r.SID AS RegSID,
    $ENTITY_REGISTRY AS Type,
    'registries' AS Plural,
    'registry' AS Singular,
    NULL AS ParentSID,
    r.SID AS eSID,
    r.UID AS UID,
    '' AS Abstract,
    '' AS Path
FROM Registries AS r

UNION SELECT                            # Gather Groups
    g.RegistrySID AS RegSID,
    $ENTITY_GROUP AS Type,
    g.Plural AS Plural,
    g.Singular AS Singular,
    g.RegistrySID AS ParentSID,
    g.SID AS eSID,
    g.UID AS UID,
    g.Abstract,
    g.Path
FROM "Groups" AS g

UNION SELECT                    # Add Resources
    r.RegistrySID AS RegSID,
    $ENTITY_RESOURCE AS Type,
    r.Plural AS Plural,
    r.Singular AS Singular,
    r.GroupSID AS ParentSID,
    r.SID AS eSID,
    r.UID AS UID,
    r.Abstract,
    r.Path
FROM Resources AS r

UNION SELECT                    # Add Metas
    metas.RegistrySID AS RegSID,
    $ENTITY_META AS Type,
    'metas' AS Plural,
    'meta' AS Singular,
    metas.ResourceSID AS ParentSID,
    metas.SID AS eSID,
    'meta',
    metas.Abstract,
    metas.Path
FROM Metas AS metas

UNION SELECT                    # Add Versions for non-xref Resources
    v.RegistrySID AS RegSID,
    $ENTITY_VERSION AS Type,
    'versions' AS Plural,
    'version' AS Singular,
    v.ResourceSID AS ParentSID,
    v.SID AS eSID,
    v.UID AS UID,
    v.Abstract,
    v.Path
FROM Versions AS v

UNION SELECT                    # Add Versions for xref Resources
    v.RegistrySID AS RegSID,
    $ENTITY_VERSION AS Type,
    'versions' AS Plural,
    'version' AS Singular,
    m.ResourceSID AS ParentSID,
    CONCAT('-', m.ResourceSID, '-', v.SID) AS eSID,
    v.UID AS UID,
    CONCAT(sR.Abstract, ',versions') AS Abstract,
    CONCAT(sR.Path, '/versions/', v.UID) AS Path
FROM Metas AS m
JOIN Versions AS v ON (v.ResourceSID=m.xRefSID)
JOIN Resources AS sR ON (sR.SID=m.ResourceSID)
WHERE m.xRefSID IS NOT NULL ;

CREATE TABLE ResourceContents (
    VersionSID      VARCHAR(255),
    Content         MEDIUMBLOB,

    PRIMARY KEY (VersionSID)
);

# This pulls-in or creates all props in Resources due to default Ver processing
CREATE VIEW DefaultProps AS
SELECT                            # Get default prop for non-xref resources
    p.RegistrySID,
    m.ResourceSID AS EntitySID,
    p.PropName,
    p.PropValue,
    p.PropType,
    false                          # DocView
FROM Metas AS m
JOIN Versions AS v
  ON (m.ResourceSID=v.ResourceSID AND v.UID=m.defaultVID)
JOIN Props AS p ON (p.EntitySID=v.SID)
WHERE m.xRefSID IS NULL

UNION SELECT                       # Get default prop for xref resources
    p.RegistrySID,
    m.ResourceSID AS EntitySID,
    p.PropName,
    p.PropValue,
    p.PropType,
    false                          # DocView
FROM Metas AS m
JOIN Versions AS v
  ON (
    m.xRefSID=v.ResourceSID AND
        v.UID=(SELECT defaultVID FROM Metas WHERE ResourceSID=m.xRefSID)
  )
JOIN Props AS p ON (p.EntitySID=v.SID)
WHERE m.xRefSID IS NOT NULL

UNION SELECT                    # Add Resource.isdefault, always 'true'
    m.RegistrySID,
    m.ResourceSID,
    'isdefault$DB_IN',
    'true',
    'boolean',
    false                       # DocView
FROM Metas AS m ;

CREATE VIEW AllProps AS
SELECT                          # Base props
    RegistrySID,
    EntitySID,
    PropName,
    PropValue,
    PropType,
    DocView
FROM Props

UNION SELECT                    # Add Props for xRef resources
    mS.RegistrySID AS RegistrySID,
    mS.SID AS EntitySID,
    p.PropName AS PropName,
    p.PropValue AS PropValue,
    p.PropType AS PropType,
    false AS DocView
FROM Metas AS mS
JOIN Metas AS mT ON (mT.ResourceSID=mS.xRefSID)
JOIN Props AS p ON (p.EntitySID=mT.SID AND
       p.PropName NOT IN ('xref$DB_IN',CONCAT(mT.Singular,'id$DB_IN')))
WHERE mS.xRefSID IS NOT NULL

UNION SELECT                   # Add Version props for xRef resources
    mS.RegistrySID AS RegistrySID,
    CONCAT('-', mS.ResourceSID, '-', p.EntitySID) AS EntitySID,
    p.PropName AS PropName,
    p.PropValue AS PropValue,
    p.PropType AS PropType,
    false AS DocView
FROM Metas as mS
JOIN Props as p ON (p.EntitySID IN (
       SELECT eSID FROM Entities WHERE ParentSID=mS.xRefSID AND
                                       Type=$ENTITY_VERSION
     ) AND p.PropName<>'xref$DB_IN')
WHERE mS.xRefSID IS NOT NULL

UNION SELECT * FROM DefaultProps

UNION SELECT                    # Add Version.isdefault, which is calculated
  v.RegSID,
  v.eSID,
  'isdefault$DB_IN',
  IF(
      (m.defaultVID IS NOT NULL AND v.UID=m.defaultVID) OR
      (m.defaultVID IS NULL AND m.xRefSID IS NOT NULL AND
        v.UID=(SELECT defaultVID FROM Metas WHERE ResourceSID=m.xRefSID)
      ),
      'true', 'false'
    ),
  'boolean',                    # Type
  IF(LEFT(v.eSID,1)='-',false,true)  # DocView,Lie if it's not xref'd prop/ver
FROM Entities AS v
JOIN Metas AS m ON (m.ResourceSID=v.ParentSID AND v.Type='$ENTITY_VERSION')

UNION SELECT                   # Add *.xid, which is calculated
  e.RegSID,
  e.eSID,
  'xid$DB_IN',
  CONCAT('/', e.Path),
  'string',
  IF(LEFT(e.eSID,1)='-',false,true)   # A bit of a lie for DocView mode
FROM Entities AS e

UNION SELECT                   # Add in Version.RESOURCEid, which is calculated
  v.RegSID,
  v.eSID,
  CONCAT(r.Singular, 'id$DB_IN'),
  r.UID,
  'string',
  IF(LEFT(v.eSID,1)='-',false,true)  # Lie if it's not an xref'd prop/ver
FROM Entities AS v
JOIN Resources AS r ON (r.SID=v.ParentSID)
WHERE v.Type=$ENTITY_VERSION;

CREATE VIEW FullTree AS
SELECT
    e.RegSID,
    e.Type,
    e.Plural,
    e.Singular,
    e.ParentSID,
    e.eSID,
    e.UID,
    e.Path,
    p.PropName,
    p.PropValue,
    p.PropType,
    e.Abstract,
    p.DocView
FROM Entities AS e
JOIN AllProps AS p ON (p.EntitySID=e.eSID)
ORDER by Path, PropName;

CREATE VIEW Leaves AS
SELECT eSID FROM Entities
WHERE eSID NOT IN (
    SELECT DISTINCT ParentSID FROM Entities WHERE ParentSID IS NOT NULL
);

# Just for debugging purposes
CREATE VIEW VerboseProps AS
SELECT
    p.RegistrySID,
    p.EntitySID,
    e.Abstract,
    e.Path,
    p.PropName,
    p.PropValue,
    p.PropType
FROM Props as p
JOIN Entities as e ON (e.eSID=p.EntitySID)
ORDER by Path ;

# Find all of the versions of a resource. Users of this should order
# the results: ORDER BY Pos ASC, Time ASC, VersionUID ASC
# to get oldest first, newest last.
# Pos (postion) makes sure roots are first, leaves are last.
# For similar rows, order by createdat timestamps and then versionIDs
CREATE VIEW VersionAncestors AS
SELECT
    v.RegistrySID AS RegistrySID,
    v.ResourceSID AS ResourceSID,
    v.SID AS VersionSID,
    v.UID AS VersionUID,
    v.Ancestor AS Ancestor,
    v.CreatedAt AS Time,
    CASE
        WHEN v.UID=v.Ancestor THEN '0-root'
        WHEN EXISTS(SELECT 1 FROM Versions AS v2 WHERE
                    v2.ResourceSID=v.ResourceSID AND v2.Ancestor=v.UID)
             THEN '1-middle'
        ELSE '2-leaf'
    END AS Pos
FROM Versions AS v ;

# Find all Versions that are part of circular references (circles)
# Would this be better to do in code and use args(?) for regSID?
CREATE VIEW VersionCircles AS
WITH RECURSIVE cte (RegistrySID,ResourceSID,UID) AS
(
    # Start with the roots and leaves, they can never be part of a circle
    SELECT v.RegistrySID,v.ResourceSID,v.UID FROM Versions AS v
    WHERE v.Ancestor=UID OR
        NOT EXISTS(SELECT 1 FROM Versions AS v2 WHERE
                   v2.RegistrySID=v.RegistrySID AND
                   v2.ResourceSID=v.ResourceSID AND
                   v2.Ancestor=v.UID)
    UNION
    # Now find all Versions whose Ancestor is in cte
    SELECT v3.RegistrySID,v3.ResourceSID,v3.UID FROM Versions AS v3
    INNER JOIN cte ON (
        v3.RegistrySID=cte.RegistrySID AND
        v3.ResourceSID=cte.ResourceSID AND
        v3.Ancestor=cte.UID )
)
# And finally, return all Version UID that are NOT in cte (these are circular)
SELECT v.RegistrySID, v.ResourceSID, v.UID FROM Versions AS v
WHERE NOT EXISTS(SELECT 1 FROM cte
                 WHERE cte.RegistrySID=v.RegistrySID AND
                       cte.ResourceSID=v.ResourceSID AND
                       cte.UID=v.UID);
