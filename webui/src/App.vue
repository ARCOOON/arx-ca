<script setup lang="ts">
import { RouterView } from 'vue-router'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/sonner'
</script>

<template>
  <TooltipProvider :delay-duration="200">
    <RouterView />
    <Toaster position="bottom-right" :rich-colors="true" />
  </TooltipProvider>
</template>
