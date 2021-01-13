var apiURL = "localhost:4006";

var user = new Vue({
    el: '#avatar',
    data: {
        userProfilePic: null,
        name: null,
        showName: true,
        peerID: null,
        channels: [
            "General"
        ]
    },
    methods: {
        selectAvatar: function (event) {
            var input = document.createElement('input');
            input.type = 'file';

            input.onchange = e => {
                var file = e.target.files[0];

                var reader = new FileReader();
                var baseString;
                reader.onloadend = function () {
                    baseString = reader.result;
                    axios.post('http://' + apiURL + '/v1/ob/avatar', {
                        avatar: baseString.replace("data:image/jpeg;base64,", "")
                    })
                        .then(response => {
                            user.userProfilePic = 'http://' + apiURL + '/v1/ob/image/' + response.data.small;
                        })
                        .catch(error => {
                            console.log(error);
                        });
                };
                reader.readAsDataURL(file);
            };

            input.click();
        },
        updateProfileName: function(event) {
            let name = this.$refs.nameInputField.value;

            axios.get('http://' + apiURL + '/v1/ob/profile')
                .then(function (response) {
                    response.data.name = name;
                    user.peerID = response.data.peerID;
                    axios.put('http://' + apiURL + '/v1/ob/profile', response.data)
                        .catch(error => {
                            console.log(error);
                        });
                })
                .catch(error => {
                    axios.post('http://' + apiURL + '/v1/ob/profile', {
                        name: name
                    })
                        .then(response => {
                            user.profile = {name: name}
                        })
                        .catch(error => {
                            console.log(error);
                        });
                });

            user.name = name;
            user.showName = true;
        },
        toggleNameInput: function (event) {
            user.showName = !user.showName;
        },
    },
    beforeMount(){
        axios.get('http://' + apiURL + '/v1/ob/profile')
            .then(function (response) {
                if (response.data.avatarHashes) {
                    user.userProfilePic = 'http://' + apiURL + '/v1/ob/image/' + response.data.avatarHashes.small;
                    user.name = response.data.name;
                    user.peerID = response.data.peerID;
                }
            })
            .catch(error => {
                console.log(error);
            });
    },
});

var channels = new Vue({
    el: '#channelList',
    data: {
        channels: [
            {name: "General", id:"channel-General", active:true}
        ]
    },
    methods: {
        addToChannelList: function (event) {
            let channel = this.$refs.channelInputField.value;
            document.getElementById('channelModal').style.display='none';
            for (var i in channels.channels) {
                channels.channels[i].active = false;
                if (channels.channels[i].name == channel) {
                    channels.changeActiveChannel(channels.channels[i].id);
                    return
                }
            }
            channels.channels.push({name: channel, id: "channel-"+ channel, active:true});
            axios.post('http://' + apiURL + '/v1/ob/openchannel/' + channel.toLowerCase())
                .catch(error => {
                    console.log(error);
                });
        },
        changeActiveChannel: function (selected) {
            for (var i in channels.channels) {
                channels.channels[i].active = channels.channels[i].id == selected;
            }
        }
    },
    beforeMount(){
        axios.post('http://' + apiURL + '/v1/ob/openchannel/general')
            .catch(error => {
                console.log(error);
            });
    }
});

var messages = new Vue({
    el: '#msgArea',
    data: {
        messages: [],
    },
    methods: {
        loadChannelMessages: function(channel) {
            this.messages = [];
            var currentDate = new Date();
            axios.get('http://' + apiURL + '/v1/ob/channelmessages/'+channel)
                .then(function (msgResponse) {
                    for (var i in msgResponse.data) {
                        var messageTimestamp = new Date(msgResponse.data[i].timestamp);
                        var formattted = "";
                        if (messageTimestamp.getDate() == currentDate.getDate()) {
                            formattted = formatAMPM(messageTimestamp);
                        } else {
                            formattted = formatAMPM(messageTimestamp) + " " + messageTimestamp.toDateString();
                        }
                        messages.messages.unshift({
                            name: msgResponse.data[i].peerID,
                            avatar: "765-default-avatar.png",
                            formattedTime: formattted,
                            fromMe: false,
                            message: msgResponse.data[i].message,
                            peerID: msgResponse.data[i].peerID,
                        });

                        (function(peerID) {
                            axios.get('http://' + apiURL + '/v1/ob/profile/' + peerID)
                                .then(function (profileResponse) {
                                    for (var i in messages.messages) {
                                        if (messages.messages[i].peerID == peerID) {
                                            messages.messages[i].name = profileResponse.data.name;
                                            messages.messages[i].avatar = 'http://' + apiURL + '/v1/ob/image/' + profileResponse.data.avatarHashes.small;
                                        }
                                    }
                                })
                                .catch(error => {
                                    console.log(error)
                                });
                        })(msgResponse.data[i].peerID)
                    }
                })
                .catch(error => {
                    console.log(error);
                });
        }
    },
    beforeMount(){
        this.loadChannelMessages("general")
    }
});

function formatAMPM(date) {
    var hours = date.getHours();
    var minutes = date.getMinutes();
    var ampm = hours >= 12 ? 'PM' : 'AM';
    hours = hours % 12;
    hours = hours ? hours : 12; // the hour '0' should be '12'
    minutes = minutes < 10 ? '0'+minutes : minutes;
    var strTime = hours + ':' + minutes + ' ' + ampm;
    return strTime;
}
