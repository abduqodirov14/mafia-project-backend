export class VoiceChat {
  constructor(userID, sendFn) {
    this.userID = userID;
    this.sendFn = sendFn;
    this.stream = null;
    this.peers = {};
    this.active = false;
  }

  async toggle() {
    this.active ? this.stop() : await this.start();
  }

  async start() {
    try {
      this.stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
      this.active = true;
    } catch {
      throw new Error('Mikrofon ruxsati yo\'q');
    }
  }

  stop() {
    this.stream?.getTracks().forEach(t => t.stop());
    this.stream = null;
    Object.values(this.peers).forEach(pc => pc.close());
    this.peers = {};
    this.active = false;
  }

  connectToPeers(players) {
    if (!this.active) return;
    for (const p of players) {
      if (p.id !== this.userID && p.is_alive) {
        this._createPeer(p.id, true);
      }
    }
  }

  async handleSignal(d) {
    if (!this.active || d.to !== this.userID) return;

    const fromID = d.from;
    if (d.type === 'offer') {
      this._createPeer(fromID, false);
      const pc = this.peers[fromID];
      await pc.setRemoteDescription(d.sdp);
      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);
      this.sendFn('voice_signal', { type: 'answer', to: fromID, from: this.userID, sdp: answer });
    } else if (d.type === 'answer') {
      this.peers[fromID]?.setRemoteDescription(d.sdp);
    } else if (d.type === 'candidate') {
      this.peers[fromID]?.addIceCandidate(d.candidate);
    }
  }

  _createPeer(remoteID, initiate) {
    if (this.peers[remoteID]) return;

    const pc = new RTCPeerConnection({
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
    });
    this.peers[remoteID] = pc;

    if (this.stream) {
      this.stream.getTracks().forEach(t => pc.addTrack(t, this.stream));
    }

    pc.ontrack = (e) => {
      const audio = document.createElement('audio');
      audio.srcObject = e.streams[0];
      audio.autoplay = true;
      document.body.appendChild(audio);
    };

    pc.onicecandidate = (e) => {
      if (e.candidate) {
        this.sendFn('voice_signal', {
          type: 'candidate', to: remoteID, from: this.userID, candidate: e.candidate,
        });
      }
    };

    if (initiate) {
      pc.createOffer().then(offer => {
        pc.setLocalDescription(offer);
        this.sendFn('voice_signal', {
          type: 'offer', to: remoteID, from: this.userID, sdp: offer,
        });
      });
    }
  }
}
