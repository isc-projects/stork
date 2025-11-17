[bug] marcin

    Fix an issue with slow loading of the dialog box displaying
    the zone RRs. The issue was initially observed on Safari, with
    loading times reaching tens of seconds. However, it could also
    take several seconds on other browsers. The new solution uses
    a virtual scroller. It significantly improves the dialog box
    loading time on all browsers.
    (Gitlab #2096)
