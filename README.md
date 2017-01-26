# keto

Keto is under development and should not be used or treated as working
software. Everything will probably change.

Currently, only `fake` cloud is implemented, which can be used for getting user
workflows right first.


### Building

```
$ go get -u github.com/UKHomeOffice/keto/cmd/keto
```


### Fake cloud persistence

```
$ export KETO_FAKE_STATE_FILE=/keto_fake_cloud_state.json
$ keto --cloud=fake create nodepool master --cluster=foo
$ keto --cloud=fake get nodepool
```
