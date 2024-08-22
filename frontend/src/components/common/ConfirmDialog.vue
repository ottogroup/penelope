<script setup lang="ts">

import {PropType} from "vue";

const emits = defineEmits(['confirm', 'cancel'])

const model = defineModel({default: false, required: true})

interface Options {
  title?: string
  message?: string
  color?: string
  width?: string
  confirmButtonText?: string
  confirmButtonColor?: string
  cancelButtonText?: string
  cancelButtonColor?: string
}

defineProps({
  title: {
    type: String,
    default: '',
  },
  message: {
    type: String,
    default: '',
  },
  color: {
    type: String,
    default: 'primary',
  },
  width: {
    type: String,
    default: '500',
  },
  options: {
    type: Object as PropType<Options>,
    default: () => ({
      title: 'Are you sure?',
      message: '',
      color: 'primary',
      width: '500',
      confirmButtonText: 'OK',
      confirmButtonColor: 'primary',
      cancelButtonText: 'Cancel',
      cancelButtonColor: 'secondary',
    }),
  }
})

const cancel = () => {
  emits('cancel')
}

const confirm = () => {
  emits('confirm')
}

</script>

<template>
  <v-dialog
    v-model="model"
    :width="width"
    @keydown.esc="cancel"
  >
    <v-card>
      <v-toolbar dark :color="options.color" dense flat>
        <v-toolbar-title class="text-body-2 font-weight-bold grey--text">
          {{ options.title }}
        </v-toolbar-title>
      </v-toolbar>
      <v-card-text
        v-show="!!options.message"
        class="pa-4 black--text"
        v-html="options.message"
      ></v-card-text>
      <v-card-actions class="pt-3">
        <v-spacer></v-spacer>
        <v-btn
          class="body-2 font-weight-bold"
          @click.native="cancel"
          :text="options.cancelButtonText"
        />
        <v-btn
          color="red"
          class="body-2 font-weight-bold"
          outlined
          @click.native="confirm"
          :text="options.confirmButtonText"
        />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<style scoped>

</style>
