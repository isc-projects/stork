import tempfile

from core.lease_generators import gen_dhcp4_lease_file, gen_dhcp6_lease_file


def test_gen_dhcp4_lease_file():
    expected = ''

    with tempfile.TemporaryFile("r+t") as f:
        gen_dhcp4_lease_file(f)

        f.seek(0)
        actual = f.read()
        assert expected == actual


def test_gen_dhcp4_lease_file():
    expected = ''

    with tempfile.TemporaryFile("r+t") as f:
        gen_dhcp4_lease_file(f)

        f.seek(0)
        actual = f.read()
        assert expected == actual
