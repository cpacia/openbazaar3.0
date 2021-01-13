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
        ],
        activeChannel: "General",
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
            messages.messages = [];
            messages.loadChannelMessages(channel.toLowerCase());
            channels.activeChannel = channel;
        },
        changeActiveChannel: function (selected) {
            for (var i in channels.channels) {
                channels.channels[i].active = channels.channels[i].id == selected;
            }
            var ch = selected.replace("channel-", "");
            messages.messages = [];
            messages.loadChannelMessages(ch.toLowerCase());
            channels.activeChannel = ch;
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
        busy: false,
    },
    methods: {
        loadChannelMessages: function(channel, offsetID) {
            var offset = "";
            if (offsetID != null && offsetID != "") {
                offset = "&offsetID=" + offsetID;
            }
            axios.get('http://' + apiURL + '/v1/ob/channelmessages/'+channel+"?limit=7" + offset)
                .then(function (msgResponse) {
                    for (var i in msgResponse.data) {
                        messages.setMessage(msgResponse.data[i], true)
                    }
                    messages.busy = false;
                })
                .catch(error => {
                    console.log(error);
                });
        },
        setMessage: function(message, append) {
            var currentDate = new Date();
            var messageTimestamp = new Date(message.timestamp);
            var formattted = "";
            if (messageTimestamp.getDate() == currentDate.getDate()) {
                formattted = formatAMPM(messageTimestamp);
            } else {
                formattted = formatAMPM(messageTimestamp) + " " + messageTimestamp.toDateString();
            }
            var newMessage = {
                name: "anonymous",
                avatar: "765-default-avatar.png",
                formattedTime: formattted,
                message: message.message,
                peerID: message.peerID,
                cid: message.cid,
            };
            if (append) {
                messages.messages.push(newMessage);
            } else {
                messages.messages.unshift(newMessage);
            }

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
            })(message.peerID)
        },
        sendMessage: function(event) {
            axios.post('http://' + apiURL + '/v1/ob/channelmessage', {
                topic: channels.activeChannel.toLowerCase(),
                message: document.getElementById("usermsg").value
            })
                .catch(error => {
                    console.log(error)
                });
            document.getElementById("usermsg").value = "";
        },
        loadMore () {
            messages.busy = true;
            messages.loadChannelMessages(channels.activeChannel.toLowerCase(), messages.messages[messages.messages.length - 1].cid);
        }
    },
    beforeMount(){
        this.messages = [];
        this.loadChannelMessages("general");
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

var ws = new WebSocket("ws://"+apiURL+"/ws");
ws.onmessage = function (event) {
    var msg = JSON.parse(event.data);
    if (msg.channelMessage != null) {
        messages.setMessage(msg.channelMessage, false)
    }
};

document.getElementById("msgs").onscroll = () => {
    console.log(document.getElementById("msgs").scrollTop);
    console.log(document.getElementById("msgs").scrollHeight);
    console.log(document.getElementById("msgs").offsetHeight);
    console.log(document.getElementById("msgs").clientHeight);
    let topOfWindow = document.getElementById("msgs").scrollHeight + document.getElementById("msgs").scrollTop - 25 <= document.getElementById("msgs").clientHeight;
    if (topOfWindow && !messages.busy) {
        messages.loadMore();
    }
};
