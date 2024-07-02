[bug] marcin

    Delete a subnet from shared network in Kea before deleting the
    subnet. It is a workaround for the Kea issue #3455. Before this
    change, a subnet could silently fail to delete from a Kea server
    when it belonged to a shared network.
    (Gitlab #1425)