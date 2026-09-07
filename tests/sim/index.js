ELEMENT.locale(ELEMENT.lang.en)

var subnets = []
var daemons = []
var services = []
var credentials = {
    identifier: "",
    secret: "",
    authenticationMethodId: ""
}

var app = new Vue({
    el: '#app',
    data: {
        menuIndex: 'manager',
        subnets: subnets,
        daemons: daemons,
        services: services,
        disableDHCPButtons: false,
        credentials: credentials,
    },
    created: function () {
        axios.get('/subnets').then(function (response) {
            this.subnets.push(...response.data.items)
        }).catch((err) => console.log('Error getting subnets: ', err))
        axios.get('/daemons').then(function (response) {
            this.daemons.push(...response.data.items)
        }).catch((err) => console.log('Error getting daemons: ', err))
        axios.get('/services').then(function (response) {
            this.services.push(...response.data.items)
        }).catch((err) => console.log('Error getting services: ', err))
        axios.get('/session').then(function (response) {
            this.credentials.identifier = response.data.identifier;
            this.credentials.secret = response.data.secret;
            this.credentials.authenticationMethodId = response.data.authenticationMethodId;
        }).catch((err) => console.log('Error getting session: ', err))
    },
    methods: {
        menuSelect: function (key, keyPath) {
            this.menuIndex = key;
        },
        updateSubnet: function (idx, state) {
            this.disableDHCPButtons = true
            var sn = this.subnets[idx]
            var data = {
                state: state,
                rate: sn.rate,
                clients: sn.clients
            }
            axios.put('/subnets/' + idx, data).then(function (response) {
                this.subnets.length = 0;
                this.subnets.push(...response.data.items);
            }).catch((err) => console.log('Error putting subnet: ', idx, data, err))
                .finally(() => this.disableDHCPButtons = false)
        },
        query: function (idx) {
            var daemon = this.daemons[idx]
            var data = {
                attempts: daemon.attempts,
                qname: daemon.qname,
                qtype: daemon.qtype,
                transport: daemon.transport,
                rate: daemon.rate,
                clients: daemon.clients

            }
            axios.put('/query/' + idx, data).then(function (response) {
                this.daemons.length = 0;
                this.daemons.push(...response.data.items);
            }).catch((err) => console.log('Error putting query: ', idx, data, err))
        },
        perf: function (idx, state) {
            var daemon = this.daemons[idx]
            var data = {
                state: state,
                attempts: daemon.attempts,
                qname: daemon.qname,
                qtype: daemon.qtype,
                transport: daemon.transport,
                rate: daemon.rate,
                clients: daemon.clients
            }
            axios.put('/perf/' + idx, data).then(function (response) {
                this.daemons.length = 0;
                this.daemons.push(...response.data.items);
            }).catch((err) => console.log('Error putting perf: ', idx, data, err))
        },
        updateService: function (idx, operation) {
            var sn = this.subnets[idx]
            var data = { operation: operation }
            axios.put('/services/' + idx, data).then(function (response) {
                this.services.length = 0;
                this.services.push(...response.data.items);
            }).catch((err) => console.log('Error putting service: ', idx, data, err))
        },
        updateSession: function () {
            var data = {
                identifier: this.credentials.identifier,
                secret: this.credentials.secret,
                authenticationMethodId: this.credentials.authenticationMethodId
            }
            axios.put('/session', data)
                .then(function (response) {
                    this.credentials = response.data;
                })
                .catch((err) => console.log('Error updating session: ', data, err))
        }
    }
})