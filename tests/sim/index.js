ELEMENT.locale(ELEMENT.lang.en)

var subnets = []
var applications = []
var services = []

var app = new Vue({
    el: '#app',
    data: {
        menuIndex: 'manager',
        subnets: subnets,
        applications: applications,
        services: services,
        disableDHCPButtons: false
    },
    created: function () {
        axios.get('/subnets').then(function (response) {
            this.subnets.push(...response.data.items)
        }).catch((err) => console.log('Error getting subnets: ', err))
        axios.get('/applications').then(function (response) {
            this.applications.push(...response.data.items)
        }).catch((err) => console.log('Error getting applications: ', err))
        axios.get('/services').then(function (response) {
            this.services.push(...response.data.items)
        }).catch((err) => console.log('Error getting services: ', err))
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
            var application = this.applications[idx]
            var data = {
                attempts: application.attempts,
                qname: application.qname,
                qtype: application.qtype,
                transport: application.transport
            }
            axios.put('/query/' + idx, data).then(function (response) {
                this.applications.length = 0;
                this.applications.push(...response.data.items);
            }).catch((err) => console.log('Error putting query: ', idx, data, err))
        },
        perf: function (idx, state) {
            var application = this.applications[idx]
            var data = {
                state: state,
                attempts: application.attempts,
                qname: application.qname,
                qtype: application.qtype,
                transport: application.transport
            }
            axios.put('/perf/' + idx, data).then(function (response) {
                this.applications.length = 0;
                this.applications.push(...response.data.items);
            }).catch((err) => console.log('Error putting perf: ', idx, data, err))
        },
        updateService: function (idx, operation) {
            var sn = this.subnets[idx]
            var data = {operation: operation}
            axios.put('/services/' + idx, data).then(function (response) {
                this.services.length = 0;
                this.services.push(...response.data.items);
            }).catch((err) => console.log('Error putting service: ', idx, data, err))
        }
    }
})