import re
import time
from selenium.webdriver.common.action_chains import ActionChains
import logging


def find_element(sel, element_type, element, number_of_tests=10):
    for i in range(1, number_of_tests + 1):
        try:
            if element_type == "xpath":
                el = sel.find_element_by_xpath(element)
                el.is_displayed()
                print("Element '%s' found by it's type: %s" % (element, element_type))
                return el
            elif element_type == "id":
                el = sel.find_element_by_id(element)
                el.is_displayed()
                print("Element '%s' found by it's type: %s" % (element, element_type))
                return el
            elif element_type == "name":
                el = sel.find_element_by_name(element)
                el.is_displayed()
                print("Element '%s' found by it's type: %s" % (element, element_type))
                return el
            elif element_type == "class_name":
                el = sel.find_element_by_class_name(element)
                el.is_displayed()
                print("Element '%s' found by it's type: %s" % (element, element_type))
                return el
            elif element_type == "link_text":
                el = sel.find_element_by_link_text(element)
                el.is_displayed()
                print("Element '%s' found by it's type: %s" % (element, element_type))
                return el
            elif element_type == "partial_link_text":
                el = sel.find_element_by_partial_link_text(element)
                el.is_displayed()
                print("Element '%s' found by it's type: %s" % (element, element_type))
                return el
            else:
                print("Element %s not defined" % element_type)
        except Exception:
            print(
                '!!!! Iteration no. %d: Failed to find element "%s" by type: %s'
                % (i, element, element_type)
            )
            time.sleep(1)

    # If we got here, the find failed.
    return None


def display_sleep(sel, sec=1):
    """
    Add time.sleep if test are NOT running in headless mode.
    It will make easier to spot what test is doing.
    :param sel: driver
    :param sec: number of seconds int
    """
    if (
        sel.capabilities["browserName"] == "firefox"
        and not sel.capabilities["moz:headless"]
    ):
        print(">>>> Test not in headless mode - sleep %d seconds" % sec)
        time.sleep(sec)
    else:
        # I can't run chrome tests, I don't know how to check it for headless mode TODO fix it :)
        time.sleep(1)


def check_help_text(sel, id_of_help_button, id_of_help_test, help_text):
    """
    Find help, open help window, check help text, close help window
    :param sel: driver
    :param id_of_help_button: string, id to help button
    :param id_of_help_test:  string, id to help
    :param help_text: string, help text to compare
    :return:
    """
    close_all_popup_notifications(sel)
    help_button = find_element(sel, "id", id_of_help_button)
    help_button.click()
    print("Checking help content:\n\t%s" % help_text, end="")
    assert help_text in find_element(sel, "id", id_of_help_test).text
    print(" - OK!")
    help_button.click()
    display_sleep(sel)


def check_popup_notification(sel, text_message):
    """
    Find popup notification (right top) and check it's content
    :param sel:  driver
    :param text_message: string, message that should be included in popup
    """
    el = find_element(sel, "class_name", "ui-toast-close-icon")
    assert text_message in find_element(sel, "class_name", "ui-toast-summary").text
    el.click()
    display_sleep(sel)


def close_all_popup_notifications(sel, text_message=None, counter_limit=10):
    """
    Sometime we have to close all notifications before we can move forward, by default this function will close
    10 notifications without checking it's content - just to clear the screen. We can choose to check text notification
    but function check_popup_notification is better for this.
    :param sel: driver
    :param text_message: notification text, by default we don't check it
    :param counter_limit: up to how many notifications we want to close
    :return:
    """
    counter = 0
    while counter < counter_limit:
        try:
            close_icon = sel.find_element_by_class_name("ui-toast-close-icon")
        except Exception:
            # if there is no notification - break loop
            break
        if text_message is not None:
            assert (
                text_message in find_element(sel, "class_name", "ui-toast-message").text
            )
        close_icon.click()
        display_sleep(sel)


def add_stork_agent_machine(sel, address, port=None):
    """
    Add new stork agent to system
    :param sel: driver
    :param address: address of agent
    :param port: port of an agent
    """
    find_element(sel, "id", "services").click()
    find_element(sel, "id", "machines").click()
    find_element(sel, "id", "add-new-machine").click()
    find_element(sel, "id", "machine-address").clear()
    find_element(sel, "id", "machine-address").send_keys(address)
    if port is not None:
        find_element(sel, "id", "agent-port").clear()
        find_element(sel, "id", "agent-port").send_keys(port)
    find_element(sel, "id", "add-new-machine-page").click()


def move_to_different_place(sel, element_id="small-stork-logo-img"):
    """
    Sometimes you just need to move away and back to the same place e.g. to display tooltip again
    This is easy function to move, by default it moves to Stork logo
    :param sel:
    :param element_id: string
    """
    ActionChains(sel).move_to_element(find_element(sel, "id", element_id)).perform()


def go_to_dashboard(sel):
    """
    Go to main page
    :param sel: driver
    """
    find_element(sel, "id", "small-stork-logo-img").click()


def stork_login(sel, username, password, expect=True):
    """
    Login to stork, by default it will check if default empty page of stork will be loaded,
    if you expect login to fail or you are logging into not empty stork - set to False
    :param sel: driver
    :param username: string
    :param password: string
    :param expect: bool, check if login was successful
    """
    find_element(sel, "id", "username").clear()
    find_element(sel, "id", "username").send_keys(username)
    find_element(sel, "id", "password").clear()
    find_element(sel, "id", "password").send_keys(password)
    find_element(sel, "id", "sign-in-button").click()

    if expect:
        check_phrases(
            sel,
            [
                r"Welcome to Stork!",
                r"Events",
                r"Services",
                r"Configuration",
                r" Stork is a monitoring solution for ",
            ],
        )


def check_phrases(sel, phrase_lst, expect=True):
    """
    Check if list of string are displayed on page
    :param sel: driver
    :param phrase_lst: single string or list of strings
    :param expect: set to False if anything from phrase_lst should NOT be found
    """
    display_sleep(sel)
    current_page = sel.page_source
    if not isinstance(phrase_lst, list):
        phrase_lst = [phrase_lst]
    print("Checking phrase: ")
    for phrase in phrase_lst:
        if expect:
            print("\t", phrase, end="")
            if not (re.search(phrase, current_page)):
                print(current_page)
                assert False, 'Phrase "%s" not found on displayed page' % phrase
        else:
            print("\tCAN'T INCLUDE: ", phrase, end="")
            assert not (re.search(phrase, current_page)), (
                'Phrase "%s" FOUND on displayed page against expectation' % phrase
            )
        print(" - OK! ")


def find_and_check_tooltip(
    sel,
    tooltip_text,
    element_text=None,
    xpath=None,
    element_id=None,
    tooltip_class="ui-tooltip",
    use_in_refresh=False,
):
    """
    Find element that should have tooltip displayed when you hover over. Check content of this tooltip.
    Can be used in refresh loop.
    :param sel: driver
    :param tooltip_text: part of a text that should be displayed when hovered over element
    :param element_text: text of a link that we are searching for
    :param xpath: xpath of an element that we are looking for
    :param element_id: id of an element that we are looking for
    :param tooltip_class: tooltip class, default ui-tooltip
    :param use_in_refresh: False if we just want to check tooltip content, True when it's used in refresh_until_status_turn_green
    :return: located element when use_in_refresh is False, boolen value when use_in_refresh is True
    """
    if element_text is not None:
        element = find_element(sel, "link_text", element_text)
    elif xpath is not None:
        element = find_element(sel, "xpath", xpath)
    elif element_id is not None:
        element = find_element(sel, "id", element_id)
    else:
        assert False, "you have to set element_text or xpath."
    for _ in range(7):
        try:
            ActionChains(sel).move_to_element(element).perform()
            break
        except Exception:
            logging.exception("find_and_check_tooltip() failed")
            pass
    display_sleep(sel)
    if not use_in_refresh:
        el = find_element(sel, "class_name", tooltip_class)
        assert (
            tooltip_text in el.text
        ), "Tooltip text expected: %s; but displayed: %s" % (tooltip_text, el.text)
        return element
    else:
        if tooltip_text in find_element(sel, "class_name", tooltip_class).text:
            return True
        move_to_different_place(sel)
        return False


def refresh_until_status_turn_green(function_to_find_an_element, refresh_button, sel):
    """
    This function allow us to refresh page and check if status of an daemon turned green, will stop after 60 tries.
    :param function_to_find_an_element: function that locates element, passed via lambda
    :param refresh_button: refresh button, selenium object
    :param sel: driver
    """
    status = False
    counter = 1
    while status or counter < 60:
        print("Refreshing for the %d time." % counter)
        refresh_button.click()
        display_sleep(sel)
        if function_to_find_an_element():
            time.sleep(2)
            break
        counter += 1
