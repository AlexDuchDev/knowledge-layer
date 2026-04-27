A domain module should usually include:

<domain>/
├─ models/
├─ service/
├─ repository/
├─ policy/
├─ events/
├─ dto/
└─ tests/

Not every module needs every folder, but the intent should stay clear:
models define domain objects
service contains business logic
repository contains persistence access
policy contains decision logic where relevant
events contain domain events
dto contains transport-safe contracts
tests focus on the module boundary
