[![CI](https://github.com/xregistry/server/actions/workflows/ci.yaml/badge.svg)](https://github.com/xregistry/server/actions/workflows/ci.yaml)

# xreg-github

Implementation of the [xRegistry](https://xregistry.io) spec.
A live version is available at
[https://xregistry.soaphub.org?ui](https://xregistry.soaphub.org?ui) too.

Still a long way to go.

To run the official image:
```
# You need to have Docker installed

docker run -ti -p 8080:8080 ghcr.io/xregistry/xreg-server-all
```

To build and run it locally:
```
# You need to have Docker installed

# Build, test and run the xreg server (creates a new DB each time):
$ make

or to use existing DB (no tests):
$ make start
```

Try it:
```
# In a browser go to:
  http://localhost:8080?ui

# Or just:
$ curl http://localhost:8080
$ curl http://localhost:8080?inline

# To run a mysql client to see the DBs (debugging):
$ make mysql-client
```

# Developers

See `misc/Dockefile-dev` for the minimal things you'll need to install.
Useful Makefile targets:
```
- make              : build all, test and run the server (alias for 'all')
- make all          : build all, test and run the server (reset the DB)
- make run          : build server and run it (no tests, reset the DB)
- make start        : build server and run it (no tests, do not reset the DB)
- make test         : build all, images and run tests, don't run server
- make clean        : erase all build artifacts, stop mysql. Basically, reset
- make server       : build the server
- make cmds         : build the exes (server and CLIs)
- make image        : build the all Docker images
- make push         : push the Docker images to DockerHub
- make mysql        : just start mysql as a Docker container
- make mysql-client : run the mysql client, for testing
- make testdev      : build a dev docker image, and build/test/run everything
                      to make sure the minimal dev install requirements
                      haven't changed
```

OLD TODO:
- Move the logic that takes the Path into account for the query into
  GenerateQuery
- Make sure that the Path entity is always in the result set when filtering
- twiddle the self and XXXUrls to include proper filter and inline stuff
- see if we can get rid of the recursion stuff
- should we add "/" to then end of the Path for non-collections, then
  we can just look for PATH/%  and not PATH + PATH/%
- can we set the registry's path to "" instead of NULL ?? already did, test it
- add support for boolean types (set/get/filter)

TODOs:
- add tests for multiple registries at the same time
- test filtering for (non)empty values - e.g. filter=id=  filter=id
  - empty complex types too
- test for filtering case-insensitive string compares
- test for filtering with string being part of value
- test for exact match of numerics, bools
- add complex filters testcases
- copy all of the types tests in http from Reg to groups, resources and vers
- test to make sure an ID in the body == ID in URL for reg and group
- test to ensure we can do 2 tx at the same time
- need to decide on the best tx isolation level
- add http test for maxVersions (already have non-http tests)

- see if we can prepend Path with / or append it with /
- see if we can remove all uses of JustSet except for the few testing cases
  where we need to set a property w/o verifying/saving it
  - Just see if we can clean-up the Set... stuff in general
- convert internal errors into "panic" so any "error" returned is a user error
- see if we can move "#resource??" into the attributes struct
- fix it so that a failed call to model.Save() (e.g. verify fails) doesn't
  invalidate existing local variables. See if we can avoid redownloading the
  model from the DB
- make sure we check for uniqueness of IDs - they're case insensitive and
  unique within the scope of their parent
- make sure we throw an error if ?specversion on HTTP requests specifies the
  wrong version

- pagination
- have DB generate the COLLECTIONcount attributes so people can query over
  them and we don't need the code to calculate them (can we due to filters?)
- support overriding spec defined attributes - like "format"
- support changing the model - test for invalid changes
- add tests for immutable attributes
- test filtering on bool attributes where they search for attr=false
- support the resource sticky/default attributes
  - remove ?setdefault.. for some apis
  - process ?setdefaultversionid flag before we update things
  - make sure we support them as http headers too
- support createdat and modifiedat as http headers
- make sure we don't let go http add "content-type" header for docs w/o a value
- add support for PUT / to update the model
- add model tests for typemap - just that we can set via full model updates
- support the DB vanishing for a while
- create an UpdateDefaultVersion func in resource.go to move it from http logic
- support ximport
- support validating that xref points to the same resource def
- allow $meta on hasdoc=false resources
- fix init.sql, it's too slow due to latest xref stuff in commit 9c583e7
- support ETag/If-Match
- Split the model.verify stuff so it doesn't verify the data unless asked to
- add support for shortself
- see if we can create a $RESOURCEid SpecProp for Version&Meta level and then
  use "$SINGULRid" for everything including Versions, but not Meta
- test that we can't set defaultversionsticky when setdefaultversionsticky=false
- add more tests around defaultversionid/sticky to cover all the variants.
  - and include xref twiddling
  - verify TestHTTPDelete epoch values are correct
  - verify TestHTTPNestedResources epoch values are correct
  - verify TestMetaCombos epoch values are correct
  - stuff in TestXrefRevert
- support PATCH on capabilities /cap and "cap" attr
- bump registry.epoch when capabilities change
- make meta.readonly a non-readonly attribute if the client is an admin
  - by making it mutable and changings it's meta-model
- test cap.maxversions & creating resource type that violate it
- test cap.sticky & creating resource type that violate it
- verify that attrs with default values require "serverrequired" to be true
- test default values - incuding within objects
- add tests for immutable attributes - in particular extensions
- see if we can add RESOURCEid to Versions so we don't need special logic
  to exclude them in the code (e.g. in rID's updatefn and validateobj)
- why are "capabilities" and "model" readonly?
- allow any model change, verify entities
- make sure that maxversions=0 when we only support 1 means sitcky must be false
- require "none" to be in "compats" enum
- stop defaulting the body to {}
- group/resource type names must be unique across plural and singular
- support $details when hasdoc=false
  - apparently $details is appened to the end of URLs pointing to this resourc/version in this case
- support PATCH on collections - fix testcase TestHTTPMissingBody
- test PATCH on collections - in particular versions
- verify readonly attrs are ignored on writes, but readonly resources generate an error
