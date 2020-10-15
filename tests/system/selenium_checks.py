import re
import time
from selenium.common.exceptions import ElementNotInteractableException
from selenium.webdriver.common.action_chains import ActionChains


def display_sleep(sel, sec=1):
    """
    Add time.sleep if test are NOT running in headless mode.
    It will make easier to spot what test is doing.
    :param sel: driver
    :param sec: number of seconds int
    """
    if not sel.__dict__["capabilities"]['moz:headless']:
        print('>>>> Test not in headless mode - sleep %d seconds' % sec)
        time.sleep(sec)


def check_help_text(sel, xpath_to_help_button, xpath_to_help_text, help_text):
    """
    Find help, open help window, check help text, close help window
    :param sel: driver
    :param xpath_to_help_button: string, xpath to help button
    :param xpath_to_help_text:  string, xpath to help
    :param help_text: string, help text to compare
    :return:
    """
    close_all_popup_notifications(sel)
    sel.find_element_by_xpath(xpath_to_help_button).click()
    assert help_text in sel.find_element_by_xpath(xpath_to_help_text).text
    sel.find_element_by_xpath(xpath_to_help_button).click()
    display_sleep(sel)


def check_popup_notification(sel, text_message):
    """
    Find popup notification (right top) and check it's content
    :param sel:  driver
    :param text_message: string, message that should be included in popup
    """
    sel.find_element_by_class_name('ui-toast-close-icon')
    assert text_message in sel.find_element_by_class_name('ui-toast-message').text
    sel.find_element_by_class_name('ui-toast-close-icon').click()
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
            sel.find_element_by_class_name('ui-toast-close-icon')
        except:
            # if there is no notification - break loop
            break
        if text_message is not None:
            assert text_message in sel.find_element_by_class_name('ui-toast-message').text
        sel.find_element_by_class_name('ui-toast-close-icon').click()
        display_sleep(sel)


def add_stork_agent_machine(sel, address, port=None):
    """
    Add new stork agent to system
    :param sel: driver
    :param address: address of agent
    :param port: port of an agent
    """
    sel.find_element_by_id('Services').click()
    sel.find_element_by_id('Machines').click()
    sel.find_element_by_xpath('/html/body/app-root/app-machines-page/div/div/div[2]/button[1]/span[2]').click()
    sel.find_element_by_id("machineAddress").clear()
    sel.find_element_by_id("machineAddress").send_keys(address)
    if port is not None:
        sel.find_element_by_id("agentPort").clear()
        sel.find_element_by_id("agentPort").send_keys(port)
    sel.find_element_by_id('addNewMachine').click()


def move_to_different_place(sel, xpath="/html/body/app-root/div/a/img"):
    """
    Sometimes you just need to move away and back to the same place e.g. to display tooltip again
    This is easy function to move, by default it moves to Stork logo
    :param sel:
    :param xpath:
    """
    ActionChains(sel).move_to_element(sel.find_element_by_xpath(xpath)).perform()


def go_to_dashboard(sel):
    """
    Go to main page
    :param sel: driver
    """
    sel.find_element_by_xpath("/html/body/app-root/div/a/img").click()


def stork_login(sel, username, password, expect=True):
    """
    Login to stork, by default it will check if default empty page of stork will be loaded,
    if you expect login to fail or you are logging into not empty stork - set to False
    :param sel: driver
    :param username: string
    :param password: string
    :param expect: bool, check if login was successful
    """
    sel.find_element_by_name("username").clear()
    sel.find_element_by_name("username").send_keys(username)
    sel.find_element_by_name("password").clear()
    sel.find_element_by_name("password").send_keys(password)
    sel.find_element_by_id('SignInButton').click()

    if expect:
        check_phrases(sel, [r'Welcome to Stork!', r'Events', r'Services', r'Configuration',
                            r' Stork is a monitoring solution for '])


def check_phrases(selenium, phrase_lst, expect=True):
    """
    Check if list of string are displayed on page
    :param selenium: driver
    :param phrase_lst: single string or list of strings
    :param expect: set to False if anything from phrase_lst should NOT be found
    """
    display_sleep(selenium)
    current_page = selenium.page_source
    if not isinstance(phrase_lst, list):
        phrase_lst = [phrase_lst]
    print("Checking phrase: ")
    for phrase in phrase_lst:
        print('\t', phrase, end='')
        if expect:
            assert (re.search(phrase, current_page)), "Phrase \"%s\" not found on displayed page" % phrase
            print(" - OK! ")
        else:
            assert not (re.search(phrase, current_page)), "Phrase \"%s\" FOUND on displayed page against expectation" % phrase


def find_and_check_tooltip(selenium, tooltip_text, element_text=None, xpath=None, tooltip_class='ui-tooltip', use_in_refresh=False):
    """
    Find element that should have tooltip displayed when you hover over. Check content of this tooltip.
    Can be used in refresh loop.
    :param selenium: driver
    :param tooltip_text: part of a text that should be displayed when hovered over element
    :param element_text: text of a link that we are searching for
    :param xpath: xpath of an element that we are looking for
    :param tooltip_class: tooltip class, default ui-tooltip
    :param use_in_refresh: False if we just want to check tooltip content, True when it's used in refresh_until_status_turn_green
    :return: located element when use_in_refresh is False, boolen value when use_in_refresh is True
    """
    if element_text is not None:
        element = selenium.find_element_by_link_text(element_text)
    elif xpath is not None:
        element = selenium.find_element_by_xpath(xpath)
    else:
        assert False, "you have to set element_text or xpath."
    ActionChains(selenium).move_to_element(element).perform()
    display_sleep(selenium)
    if not use_in_refresh:
        assert tooltip_text in selenium.find_element_by_class_name(tooltip_class).text,\
            "Tooltip text expected: %s; but displayed: %s" % (tooltip_text, selenium.find_element_by_class_name(tooltip_class).text)
        return element
    else:
        if tooltip_text in selenium.find_element_by_class_name(tooltip_class).text:
            return True
        move_to_different_place(selenium)
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
        print('Refreshing for the %d time.' % counter)
        refresh_button.click()
        display_sleep(sel)
        if function_to_find_an_element():
            break
        counter += 1
