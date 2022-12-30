#### Updating the generated code for standard DHCP option definitions

DHCP standard options have well-known formats defined in the RFCs. 
Stork backend and frontend are aware of these formats and use them 
to parse option data received from Kea and send updated data to Kea.
When new options are standardized, the Stork code must be updated to
recognize the new options. In that case, a developer should define new
options in the files located in this directory and use the code
generation tool to re-generate the appropriate Golang and Typescript
code.

```console
$ rake build:code_gen
$ rake gen:std_option_defs
```
