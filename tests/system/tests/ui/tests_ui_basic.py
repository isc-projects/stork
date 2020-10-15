import pytest
import random
import string
import time

import containers
from selenium.common.exceptions import ElementNotInteractableException
from selenium.webdriver.common.keys import Keys

from selenium_checks import check_phrases, find_and_check_tooltip, refresh_until_status_turn_green, display_sleep, stork_login, \
    move_to_different_place, check_help_text, go_to_dashboard, add_stork_agent_machine


def _get_test_distros(selenium):
    # this is temporary, making running UI tests on different browsers in parallel possible
    if selenium.name == 'firefox':
        # return 'centos/7', 'centos/7'
        return 'ubuntu/18.04', 'ubuntu/18.04'
    else:
        return 'centos/7', 'centos/7'


def prepare_one_server_and_agent(agent_distro, server_distro):
    s = containers.StorkServerContainer(alias=server_distro)
    a = containers.StorkAgentContainer(alias=agent_distro)

    s.setup_bg()
    a.setup_bg()
    s.setup_wait()
    a.setup_wait()

    time.sleep(3)

    return s, a


@pytest.fixture(scope="module")
def local_or_lxd(request):
    return request.config.getoption("--local-stork")


def test_login_create_user_logout_login_with_new(selenium, local_or_lxd):
    """
    Login with default credentials
    Create new user with admin rights
    Logout and login with new user
    Check if you can create new users (you shouldn't be able)
    Logout and login with default
    Change password
    Logout
    Try to login with old credentials - should fail
    Login with new credentials
    """
    agent_distro, server_distro = _get_test_distros(selenium)
    if local_or_lxd is None:
        s, a = prepare_one_server_and_agent(agent_distro, server_distro)
        selenium.get('http://localhost:%d' % s.port)

    else:
        selenium.get(local_or_lxd)

    selenium.implicitly_wait(2)
    assert "Stork" in selenium.title

    check_phrases(selenium, [r'Dashboard for', r'Copyright 2019-2020 by ISC. All Rights Reserved.'])

    # login
    stork_login(selenium, "admin", "admin")

    # go to user page
    selenium.find_element_by_id('Configuration').click()
    selenium.find_element_by_id('Users').click()
    selenium.find_element_by_id('CreateUserAccount').click()

    # create user
    login = 'admin2' + "".join(random.sample(string.ascii_lowercase, 3)) + '1'
    selenium.find_element_by_id("userlogin").send_keys(login)
    selenium.find_element_by_id("usermail").send_keys("%s@isc.org" % login)
    selenium.find_element_by_id("userfirst").send_keys(login)
    selenium.find_element_by_id("userlast").send_keys(login)
    selenium.find_element_by_id("userpassword").send_keys(login * 2)
    selenium.find_element_by_id("userpassword2").send_keys(login * 2)
    selenium.find_element_by_id("usergroup").click()
    selenium.find_element_by_xpath('/html/body/app-root/app-users-page/div/div/div/div[2]/form/p-panel/div/div[2]/div/div/div[14]/div/div[1]/p-dropdown/div/div[4]/div/ul/p-dropdownitem[3]/li').click()
    display_sleep(selenium)
    selenium.find_element_by_id('Save').click()

    # check popup message
    assert selenium.find_element_by_class_name('ui-toast-message').text == 'New user account created\nAdding new user account succeeeded'
    selenium.find_element_by_class_name('ui-toast-close-icon').click()
    time.sleep(1)
    # logout
    selenium.find_element_by_xpath('/html/body/app-root/div/p-splitbutton/div/button[1]/span[2]').click()

    # login with new user
    stork_login(selenium, login, login * 2)

    # in configurations there should not be an option to add users
    selenium.find_element_by_id('Configuration').click()
    try:
        selenium.find_element_by_id('Users').click()
    except ElementNotInteractableException:
        pass
    else:
        assert False, "Users should not be visible"

    # logout
    selenium.find_element_by_xpath('/html/body/app-root/div/p-splitbutton/div/button[1]/span[2]').click()

    # login with default acc
    stork_login(selenium, "admin", "admin")

    # go to settings
    selenium.find_element_by_xpath('/html/body/app-root/div/p-splitbutton/div/button[2]').click()
    selenium.find_element_by_xpath('/html/body/app-root/div/p-splitbutton/div/div').click()
    check_phrases(selenium, ['superadmin', '(not specified)', 'Login:', 'Email:', 'Group:'])

    # go to change password page and change it
    selenium.find_element_by_xpath('/html/body/app-root/app-profile-page/div/div[1]/app-settings-menu/p-menu/div/ul/li[3]/a').click()
    selenium.find_element_by_xpath('/html/body/app-root/app-settings-page/div/div[2]/form/p-panel/div/div[2]/div/div/div[1]/input').send_keys("admin")
    selenium.find_element_by_xpath('/html/body/app-root/app-settings-page/div/div[2]/form/p-panel/div/div[2]/div/div/div[2]/input').send_keys("adminadmin123")
    selenium.find_element_by_xpath('/html/body/app-root/app-settings-page/div/div[2]/form/p-panel/div/div[2]/div/div/div[3]/input').send_keys("adminadmin123")
    selenium.find_element_by_class_name('ui-password-panel').click()  # hide panel that shows password strength level
    selenium.find_element_by_xpath('/html/body/app-root/app-settings-page/div/div[2]/form/p-panel/div/div[2]/div/div/div[4]/button').click()
    selenium.find_element_by_class_name('ui-toast-close-icon').click()  # turn of popup about successful password change
    time.sleep(1)

    # logout
    selenium.find_element_by_xpath('/html/body/app-root/div/p-splitbutton/div/button[1]/span[2]').click()

    # login with old password and fail
    stork_login(selenium, "admin", "admin", expect=False)

    # and popup with invalid pass should show up
    assert selenium.find_element_by_xpath("/html/body/app-root/p-toast/div/p-toastitem/div/div/div/div[1]").text == 'Invalid login or password'

    # login with new password
    stork_login(selenium, "admin", "adminadmin123")
    selenium.close()


def test_add_new_machine(selenium):
    """
    Login with default credentials
    Add stork agent with Kea4 and CA running
    Check Services > Machines
    Check help
    Open add new machine window and cancel it
    Open add new machine window and add new correct
    Check Services > KeaApps for added daemons, check help
    Check links to particular daemons and tooltips (enable monitoring of Kea6, check errors)
    Check pages of all daemons and displayed there data
    Check Dashboard page
    Check HostReservations page, should display reservations from kea4 default config, and tooltip should say it's
        from config file
    Check Shared Networks page
    Check Subnets, should display kea4 subnets
    Install Kea6
    Go to Services > KeaApps, refresh until Kea6 will be green
    Check HostReservations page, should have v4 and v6 reservations, check origin, check hosts filtering
    Check Shared Networks page, should be empty
    Check Subnets, should display kea4 and kea6 subnets, check subnets filtering and protocol dropdown menu
    Services > KeaApps > DDNS, turn on monitoring of DDNS
    install kea ddns
    Go to Services > KeaApps, refresh until DDNS will be green
    Go to DHCP > Host Reservations check if filtering actually works
    Go to Services > Machines and remove first machine from the list
    Go to Dashboard and check if it's empty
    """
    # TODO change xpaths to ids where we can
    s, a = prepare_one_server_and_agent('ubuntu/18.04', 'ubuntu/18.04')
    print('http://localhost:%d' % s.port)
    selenium.get('http://localhost:%d' % s.port)

    selenium.implicitly_wait(10)
    assert "Stork" in selenium.title
    check_phrases(selenium, [r'Dashboard for', r'Copyright 2019-2020 by ISC. All Rights Reserved.'])

    stork_login(selenium, "admin", "admin")

    selenium.find_element_by_id('Services').click()
    try:
        selenium.find_element_by_id('KeaApps').click()
    except ElementNotInteractableException:
        pass
    else:
        assert False, "Kea Apps should not be visible"

    # add stork agent
    selenium.find_element_by_id('Machines').click()
    check_phrases(selenium, ["Machines can be added by clicking the", "No machines found.", "Add New Machine"])

    selenium.find_element_by_xpath('/html/body/app-root/app-machines-page/div/div/div[2]/button[1]/span[2]').click()
    selenium.find_element_by_id('cancelMachineDialog').click()

    selenium.find_element_by_xpath('/html/body/app-root/app-machines-page/div/div/div[2]/button[1]/span[2]').click()
    selenium.find_element_by_id("machineAddress").clear()
    selenium.find_element_by_id("machineAddress").send_keys(a.mgmt_ip)
    selenium.find_element_by_id('addNewMachine').click()

    check_phrases(selenium, ["%s:8080" % a.mgmt_ip, "Hostame", "Address", "Agent Version", "Memory",
                             "Used Memory", "Uptime", "Virtualization", "Last Visited"])
    check_phrases(selenium, ["Machines can be added by clicking the", "No machines found.", "Add New Machine"],
                  expect=False)

    selenium.find_element_by_id('Services').click()
    selenium.find_element_by_id('Machines').click()
    check_phrases(selenium, ["%s:8080" % a.mgmt_ip, "stork-agent-ubuntu-18-04"])

    # check help
    check_help_text(selenium, "/html/body/app-root/app-machines-page/app-breadcrumbs/div/app-help-tip/i",
                    "/html/body/app-root/app-machines-page/app-breadcrumbs/div/app-help-tip/p-overlaypanel/div/div/div/div/p",
                    "This page displays a list of all machines that have been configured in Stork. It allows adding new machines as well as removing them.")

    selenium.find_element_by_id('Services').click()
    selenium.find_element_by_id('KeaApps').click()

    check_help_text(selenium, "/html/body/app-root/app-apps-page/app-breadcrumbs/div/app-help-tip/i",
                    "/html/body/app-root/app-apps-page/app-breadcrumbs/div/app-help-tip/p-overlaypanel/div/div/div/div/p",
                    "This page displays a list of Kea Apps.")

    # check tooltip text and dhcpv4 page
    find_and_check_tooltip(selenium, "Communication with the daemon is ok.", element_text="DHCPv4").click()
    check_phrases(selenium, ["linked with:", "log4cplus", "database:", "MySQL backend", "PostgreSQL backend",
                             "Memfile backend"])

    # check tooltip text and dhcpv6 page
    selenium.find_element_by_id('Services').click()
    selenium.find_element_by_id('KeaApps').click()
    find_and_check_tooltip(selenium, "Monitoring of this daemon has been disabled. You can enable it on the daemon tab on the Kea app page.",
                           element_text="DHCPv6").click()

    check_phrases(selenium, "This daemon is currently not monitored by Stork")
    selenium.find_element_by_class_name("ui-inputswitch").click()
    check_phrases(selenium, "There is observed issue in communication with the daemon.")

    # check tooltip text and ddns page
    selenium.find_element_by_id('Services').click()
    selenium.find_element_by_id('KeaApps').click()
    find_and_check_tooltip(selenium, "Monitoring of this daemon has been disabled. You can enable it on the daemon tab on the Kea app page.",
                           element_text="DDNS").click()
    check_phrases(selenium, "This daemon is currently not monitored by Stork")

    # check tooltip text and ca page
    selenium.find_element_by_id('Services').click()
    selenium.find_element_by_id('KeaApps').click()
    find_and_check_tooltip(selenium, "Communication with the daemon is ok.", element_text="CA").click()
    check_phrases(selenium, ["linked with:", "log4cplus"])

    # check dashboard should include just kea4 data
    selenium.find_element_by_id('DHCP').click()
    selenium.find_element_by_id('Dashboard').click()
    check_phrases(selenium, "192.0.2.0/24")

    # check host reservations should include just kea4 data
    selenium.find_element_by_id('DHCP').click()
    selenium.find_element_by_id('HostReservations').click()

    check_help_text(selenium, "/html/body/app-root/app-hosts-page/app-breadcrumbs/div/app-help-tip/i",
                    "/html/body/app-root/app-hosts-page/app-breadcrumbs/div/app-help-tip/p-overlaypanel/div/div/div/div/p[1]",
                    "This page displays a list of host reservations in the network. Kea can store host reservations in either a configuration file or a database. Reservations stored in a configuration file are retrieved continuously. Kea must have a ")

    check_phrases(selenium, ["duid=01:02:03:04:05", "192.0.2.203", "192.0.2.0/24", "client-id=01:0a:0b:0c:0d:0e:0f",
                             "192.0.2.205", "192.0.2.0/24", "client-id=01:11:22:33:44:55:66", "192.0.2.202",
                             "special-snowflake", "192.0.2.0/24", "client-id=01:12:23:34:45:56:67", "192.0.2.204",
                             "192.0.2.0/24", "hw-address=1a:1b:1c:1d:1e:1f", "192.0.2.201", "192.0.2.0/24",
                             "flex-id=73:30:6d:45:56:61:4c:75:65", "192.0.2.206"])

    find_and_check_tooltip(selenium, "The server has this host specified in the configuration file.",
                           xpath="/html/body/app-root/app-hosts-page/div/div[2]/p-table/div/div/table/tbody/tr[2]/td[6]/a/sup/span")

    selenium.find_element_by_id('DHCP').click()
    selenium.find_element_by_id('SharedNetworks').click()

    check_help_text(selenium, "/html/body/app-root/app-shared-networks-page/app-breadcrumbs/div/app-help-tip/i",
                    "/html/body/app-root/app-shared-networks-page/app-breadcrumbs/div/app-help-tip/p-overlaypanel/div/div/div/div/p",
                    "This page displays a list of shared networks.")

    # check subnet should include just kea4 data
    selenium.find_element_by_id('DHCP').click()
    selenium.find_element_by_id('Subnets').click()

    check_help_text(selenium, "/html/body/app-root/app-subnets-page/app-breadcrumbs/div/app-help-tip/i",
                    "/html/body/app-root/app-subnets-page/app-breadcrumbs/div/app-help-tip/p-overlaypanel/div/div/div/div/p[1]",
                    "This page displays a list of subnets.")

    check_phrases(selenium, ["192.0.2.0/24", "192.0.2.1-192.0.2.200"])

    # install kea
    a.install_kea('kea-dhcp6')
    a.start_kea('kea-dhcp6')

    selenium.find_element_by_id('Services').click()
    selenium.find_element_by_id('KeaApps').click()

    # refresh page until stork will notice that kea6 is up
    refresh_until_status_turn_green(lambda: find_and_check_tooltip(selenium, "Communication with the daemon is ok.",
                                                                   element_text="DHCPv6", use_in_refresh=True),
                                    selenium.find_element_by_id('Refresh'), selenium)
    # check kea6 data
    find_and_check_tooltip(selenium, "Communication with the daemon is ok.", element_text="DHCPv6").click()
    check_phrases(selenium, ["linked with:", "log4cplus", "database:", "MySQL backend", "PostgreSQL backend",
                             "Memfile backend"])
    check_phrases(selenium, "This daemon is currently not monitored by Stork. ", expect=False)

    # check dashboard should include kea4 and kea6 data
    selenium.find_element_by_id('DHCP').click()
    selenium.find_element_by_id('Dashboard').click()
    check_phrases(selenium, "192.0.2.0/24")
    check_phrases(selenium, "2001:db8:1::/64")

    # check host reservations should include kea4 and kea6
    selenium.find_element_by_id('DHCP').click()
    selenium.find_element_by_id('HostReservations').click()
    check_phrases(selenium, ["duid=01:02:03:04:05", "192.0.2.203", "192.0.2.0/24", "client-id=01:0a:0b:0c:0d:0e:0f",
                             "192.0.2.205", "192.0.2.0/24", "client-id=01:11:22:33:44:55:66", "192.0.2.202",
                             "special-snowflake", "192.0.2.0/24", "client-id=01:12:23:34:45:56:67", "192.0.2.204",
                             "192.0.2.0/24", "hw-address=1a:1b:1c:1d:1e:1f", "192.0.2.201", "192.0.2.0/24",
                             "flex-id=73:30:6d:45:56:61:4c:75:65", "192.0.2.206"])

    find_and_check_tooltip(selenium, "The server has this host specified in the configuration file.",
                           xpath="/html/body/app-root/app-hosts-page/div/div[2]/p-table/div/div/table/tbody/tr[2]/td[6]/a/sup/span")

    check_phrases(selenium, ["hw-address=00:01:02:03:04:05", "2001:db8:1::101", "duid=01:02:03:04:05:06:07:08:09:0a",
                             "2001:db8:1:0:cafe::1", "2001:db8:2:abcd::/64", "foo.example.com",
                             "duid=01:02:03:04:05:0a:0b:0c:0d:0e", "2001:db8:1::100",
                             "flex-id=73:6f:6d:65:76:61:6c:75:65", "2001:db8:1:0:cafe::2", "2001:db8:1::/64"])

    find_and_check_tooltip(selenium, "The server has this host specified in the configuration file.",
                           xpath="/html/body/app-root/app-hosts-page/div/div[2]/p-table/div/div/table/tbody/tr[10]/td[6]/a/sup/span")

    # input 192 to hosts filter, v4 should be visible and v6 should not!
    selenium.find_element_by_xpath("/html/body/app-root/app-hosts-page/div/div[1]/span/input").send_keys('192')

    check_phrases(selenium, ["hw-address=00:01:02:03:04:05", "2001:db8:1::101", "duid=01:02:03:04:05:06:07:08:09:0a",
                             "2001:db8:1:0:cafe::1", "2001:db8:2:abcd::/64", "foo.example.com",
                             "duid=01:02:03:04:05:0a:0b:0c:0d:0e", "2001:db8:1::100",
                             "flex-id=73:6f:6d:65:76:61:6c:75:65", "2001:db8:1:0:cafe::2", "2001:db8:1::/64"],
                  expect=False)

    selenium.find_element_by_xpath("/html/body/app-root/app-hosts-page/div/div[1]/span/input").clear()
    move_to_different_place(selenium)
    # clear hosts filter
    # somehow sending just enter does not work
    selenium.find_element_by_xpath("/html/body/app-root/app-hosts-page/div/div[1]/span/input").send_keys(" " + Keys.ENTER)
    # v6 hosts should be visible again
    check_phrases(selenium, ["hw-address=00:01:02:03:04:05", "2001:db8:1::101", "duid=01:02:03:04:05:06:07:08:09:0a",
                             "2001:db8:1:0:cafe::1", "2001:db8:2:abcd::/64", "foo.example.com",
                             "duid=01:02:03:04:05:0a:0b:0c:0d:0e", "2001:db8:1::100",
                             "flex-id=73:6f:6d:65:76:61:6c:75:65", "2001:db8:1:0:cafe::2", "2001:db8:1::/64"])

    # check subnet should include kea4 and kea6
    selenium.find_element_by_id('DHCP').click()
    selenium.find_element_by_id('Subnets').click()
    check_phrases(selenium, ["192.0.2.0/24", "192.0.2.1-192.0.2.200",
                             "2001:db8:1::/64", "2001:db8:1::-2001:db8:1::ffff:ffff:ffff"])

    # check subnet filter box
    # input 192 to subnet filter, v4 should be visible and v6 should not!
    selenium.find_element_by_xpath("/html/body/app-root/app-subnets-page/div/div[1]/span[1]/input").send_keys('192')
    check_phrases(selenium, ["192.0.2.0/24", "192.0.2.1-192.0.2.200"])
    check_phrases(selenium, ["2001:db8:1::/64", "2001:db8:1::-2001:db8:1::ffff:ffff:ffff"], expect=False)
    selenium.find_element_by_xpath("/html/body/app-root/app-subnets-page/div/div[1]/span[1]/input").clear()
    # input 2001 to subnet filter, v6 should be visible and v4 should not!
    selenium.find_element_by_xpath("/html/body/app-root/app-subnets-page/div/div[1]/span[1]/input").send_keys('2001')
    check_phrases(selenium, ["192.0.2.0/24", "192.0.2.1-192.0.2.200"], expect=False)
    check_phrases(selenium, ["2001:db8:1::/64", "2001:db8:1::-2001:db8:1::ffff:ffff:ffff"])

    selenium.find_element_by_xpath("/html/body/app-root/app-subnets-page/div/div[1]/span[1]/input").clear()
    move_to_different_place(selenium)
    # clear subnets filter
    # somehow sending just enter does not work
    selenium.find_element_by_xpath("/html/body/app-root/app-subnets-page/div/div[1]/span[1]/input").send_keys(" " + Keys.ENTER)
    check_phrases(selenium, ["192.0.2.0/24", "192.0.2.1-192.0.2.200",
                             "2001:db8:1::/64", "2001:db8:1::-2001:db8:1::ffff:ffff:ffff"])

    # check protocol dropdown menu
    # check ipv4
    selenium.find_element_by_xpath("/html/body/app-root/app-subnets-page/div/div[1]/span[2]/p-dropdown/div").click()
    selenium.find_element_by_xpath("/html/body/app-root/app-subnets-page/div/div[1]/span[2]/p-dropdown/div/div[4]/div/ul/p-dropdownitem[2]/li").click()
    check_phrases(selenium, ["192.0.2.0/24", "192.0.2.1-192.0.2.200"])
    check_phrases(selenium, ["2001:db8:1::/64", "2001:db8:1::-2001:db8:1::ffff:ffff:ffff"], expect=False)
    # check ipv6
    selenium.find_element_by_xpath("/html/body/app-root/app-subnets-page/div/div[1]/span[2]/p-dropdown/div").click()
    selenium.find_element_by_xpath("/html/body/app-root/app-subnets-page/div/div[1]/span[2]/p-dropdown/div/div[4]/div/ul/p-dropdownitem[3]/li").click()
    check_phrases(selenium, ["192.0.2.0/24", "192.0.2.1-192.0.2.200"], expect=False)
    check_phrases(selenium, ["2001:db8:1::/64", "2001:db8:1::-2001:db8:1::ffff:ffff:ffff"])
    # check any
    selenium.find_element_by_xpath("/html/body/app-root/app-subnets-page/div/div[1]/span[2]/p-dropdown/div").click()
    selenium.find_element_by_xpath("/html/body/app-root/app-subnets-page/div/div[1]/span[2]/p-dropdown/div/div[4]/div/ul/p-dropdownitem[1]/li").click()
    check_phrases(selenium, ["192.0.2.0/24", "192.0.2.1-192.0.2.200",
                             "2001:db8:1::/64", "2001:db8:1::-2001:db8:1::ffff:ffff:ffff"])

    # install kea ddns
    selenium.find_element_by_id('Services').click()
    selenium.find_element_by_id('KeaApps').click()
    find_and_check_tooltip(selenium, "Monitoring of this daemon has been disabled. You can enable it on the daemon tab on the Kea app page.",
                           element_text="DDNS").click()

    check_phrases(selenium, "This daemon is currently not monitored by Stork")
    selenium.find_element_by_class_name("ui-inputswitch").click()
    check_phrases(selenium, "There is observed issue in communication with the daemon.")

    a.install_kea('kea-dhcp-ddns')
    a.start_kea('kea-dhcp-ddns')
    selenium.find_element_by_id('Services').click()
    selenium.find_element_by_id('KeaApps').click()

    # refresh page until stork will notice that ddns is up
    refresh_until_status_turn_green(lambda: find_and_check_tooltip(selenium, "Communication with the daemon is ok.",
                                                                   element_text="DDNS", use_in_refresh=True),
                                    selenium.find_element_by_id('Refresh'), selenium)

    selenium.find_element_by_id('DHCP').click()
    selenium.find_element_by_id('HostReservations').click()

    check_phrases(selenium, ["hw-address=00:01:02:03:04:05", "2001:db8:1::101", "duid=01:02:03:04:05:06:07:08:09:0a",
                             "2001:db8:1:0:cafe::1", "2001:db8:2:abcd::/64", "foo.example.com",
                             "duid=01:02:03:04:05:0a:0b:0c:0d:0e", "2001:db8:1::100",
                             "flex-id=73:6f:6d:65:76:61:6c:75:65", "2001:db8:1:0:cafe::2", "2001:db8:1::/64"])

    selenium.find_element_by_id('Services').click()
    selenium.find_element_by_id('Machines').click()
    selenium.find_element_by_xpath("/html/body/app-root/app-machines-page/div/p-table/div/div/table/tbody/tr[1]/td[12]/button").click()
    selenium.find_element_by_xpath("/html/body/app-root/app-machines-page/div/p-menu/div/ul/li[2]/a").click()
    check_phrases(selenium, ["Machines can be added by clicking the", "No machines found.", "Add New Machine"])
    go_to_dashboard(selenium)
    check_phrases(selenium, r'Welcome to Stork!')

    selenium.close()

# TODO let's finish this
# def test_recreate_demo(selenium):
#     """
#     Recreate demo.
#     """
#     server = containers.StorkServerContainer(alias='centos/7')
#     agent_bind_9 = containers.StorkAgentContainer(alias='ubuntu/18.04')
#
#     server.setup_bg()
#     agent_bind_9.setup_bg()
#     server.setup_wait()
#     agent_bind_9.setup_wait()
#
#     time.sleep(3)
#
#     agent_bind_9.install_bind(conf_file='../../docker/named.conf')
#     agent_bind_9.uninstall_kea()
#     selenium.get('http://localhost:%d' % server.port)
#
#     selenium.implicitly_wait(10)
#     assert "Stork" in selenium.title
#     check_phrases(selenium, [r'Dashboard for', r'Copyright 2019-2020 by ISC. All Rights Reserved.'])
#     stork_login(selenium, "admin", "admin")
#
#     selenium.find_element_by_xpath('/html/body/app-root/div/p-splitbutton/div/button[2]').click()
#     selenium.find_element_by_xpath('/html/body/app-root/div/p-splitbutton/div/div').click()
#     check_phrases(selenium, ['superadmin', '(not specified)', 'Login:', 'Email:', 'Group:'])
#
#     add_stork_agent_machine(selenium, agent_bind_9.mgmt_ip)
#
#     check_phrases(selenium, ["%s:8080" % agent_bind_9.mgmt_ip, "Hostame", "Address", "Agent Version", "Memory",
#                              "Used Memory", "Uptime", "Virtualization", "Last Visited", "BIND 9 App"])
#     check_phrases(selenium, ["Machines can be added by clicking the", "No machines found.", "Add New Machine"],
#                   expect=False)
#
#     selenium.find_element_by_id('Services').click()
#     selenium.find_element_by_id('BIND9Apps').click()
#
#     refresh_until_status_turn_green(lambda: find_and_check_tooltip(selenium, "Communication with the daemon is ok.",
#                                                                    element_text="named", use_in_refresh=True),
#                                     selenium.find_element_by_id('Refresh'), selenium)
