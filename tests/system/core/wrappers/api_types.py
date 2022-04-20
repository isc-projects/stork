

from typing import List, TypedDict


class User(TypedDict):
    id: int
    email: str
    groups: List[int]
    lastname: str
    login: str
    name: str


class UserList(TypedDict):
    items: List[User]
    total: int


class UserCreate(TypedDict):
    user: User
    password: str


class Group(TypedDict):
    name: str


class GroupList(TypedDict):
    items: List[Group]
    total: int


class Machine(TypedDict):
    id: int
    address: str
    agentPort: str
    authorized: bool


class MachineList(TypedDict):
    items: List[Machine]
    total: int


class AccessPoint(TypedDict):
    address: str


class App(TypedDict):
    version: str
    accessPoints: List[AccessPoint]
    type: str


class MachineState(Machine):
    agentToken: str
    agentVersion: str
    apps: List[App]
    cpus: int
    lastVisitedAt: str


class Subnet(TypedDict):
    pass


class SubnetList(TypedDict):
    items: List[Subnet]
    total: int


class Event(TypedDict):
    createdAt: str
    text: str
    details: str


class EventList(TypedDict):
    items: List[Event]
    total: int


class Lease(TypedDict):
    ipAddress: str
    state: int


class LeaseList(TypedDict):
    items: List[Lease]
    total: int
    conflicts: List
