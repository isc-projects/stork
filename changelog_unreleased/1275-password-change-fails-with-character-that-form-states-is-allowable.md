[bug] piotrek

    Fixed a bug in UI of the password change form.
    The problem was when user provided New password containing special
    characters e.g. +. Even though New password and Confirm password
    where identical, form validation was failing and user could not
    submit New password change form. Similar issues could be
    experienced when New user account was being created or existing user
    account being edited by an admin. The issue there was also fixed.
    (Gitlab #1275)
