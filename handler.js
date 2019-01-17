'use strict';

const line = require('@line/bot-sdk');
const client = new line.Client({ channelAccessToken: process.env.LINE_ACCESS_TOKEN });
const crypto = require('crypto');
const ua = require('superagent');

const SawayakaStores = [
  'KR00398061', // 函南
  'KR00299583', // 沼津学園通り
];

async function getSawayakaStoreStatus() {
  const sawayaka = await ua.get('https://airwait.jp/WCSP/api/external/stateless/store/getWaitInfo')
    .query({
      limit: 50,
      key: 'UZTa9O6QvHM1vtyLpxcqNyUlbfuT0DYJ',
      domain: 'www.genkotsu-hb.com',
      storeId: SawayakaStores.join(','),
      timestamp: new Date().getTime(),
    })
    .set('Origin', 'https://www.genkotsu-hb.com')

  //console.log("API GET:", sawayaka.status);

  const body = JSON.parse(sawayaka.text);
  const ret = [];

  return body.innerDto.stores.map(s => {
    s.storeName = s.storeName.replace('さわやか', '');
    return s;
  })
}
 
module.exports.callback = async (event, context) => {
  try {
    const signature = crypto.createHmac('sha256', process.env.LINE_CHANNEL_SECRET).update(event.body).digest('base64');
    const checkHeader = (event.headers || {})['X-Line-Signature'];

    if (signature !== checkHeader) {
      console.log("signature check failed");

      return context.succeed({
        statusCode: 403,
      });
    }

    const body = JSON.parse(event.body);
    const mess = body.events[0];

    if (mess.replyToken === '00000000000000000000000000000000') {
      // for line's connection test
      return context.succeed({ statusCode: 200 });
    }

    const text = mess.message.text;

    if (text !== 'ゅびぃ、さわやかいきたい')    {
      return context.succeed({ statusCode: 200 });
    }

    const storeStatuses = await getSawayakaStoreStatus();
    const sendMessage = [];

    sendMessage.push(
      'ルビィのために沼津近辺のさわやかの混雑状況を調べてきましたわ。',
      '',
      storeStatuses.map(s => `${s.storeName}は ${s.waitCount}組で${s.waitTime}待ち`).join('、'),
      'ですわ。'
    );

    await client.replyMessage(mess.replyToken, {
      'type': 'text',
      'text': sendMessage.join("\n"),
    })

    console.log("ぅゅ！");
    return context.succeed({ statusCode: 200 });

  } catch (e) {
    console.log("Error: ", e);
    return context.succeed({ statusCode: 200 });
  }
};
