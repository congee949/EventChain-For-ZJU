import { createApp } from 'vue';
import { createPinia } from 'pinia';
import {
  ElButton,
  ElDatePicker,
  ElEmpty,
  ElInput,
  ElInputNumber,
  ElOption,
  ElSelect,
} from 'element-plus';
import 'element-plus/es/components/base/style/css';
import 'element-plus/es/components/button/style/css';
import 'element-plus/es/components/date-picker/style/css';
import 'element-plus/es/components/empty/style/css';
import 'element-plus/es/components/input/style/css';
import 'element-plus/es/components/input-number/style/css';
import 'element-plus/es/components/message/style/css';
import 'element-plus/es/components/option/style/css';
import 'element-plus/es/components/select/style/css';
import App from './App.vue';
import router from './router/index.js';
import './assets/styles/variables.css';
import './assets/styles/global.css';

const app = createApp(App);

app.use(createPinia());
app.use(router);

for (const component of [ElButton, ElDatePicker, ElEmpty, ElInput, ElInputNumber, ElOption, ElSelect]) {
  app.component(component.name, component);
}

app.mount('#app');
