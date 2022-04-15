

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
