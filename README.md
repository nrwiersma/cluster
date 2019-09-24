# Clustering Example

Very very WIP on Raft + Serf clustering, just for fun an profit

### References

Referencing the following to learn how to build reliable clusters:

* https://github.com/travisjeffery/jocko
* https://github.com/hashicorp/consul/tree/master/agent/consul

### Goals

* [ ] Be able to cluster in a stable and recoverable way
* [ ] Decent API to deal with that includes querying the store and getting store events.
* [ ] Be able to automatically forward requests to the leader
* [ ] mTLS encrypt all connections