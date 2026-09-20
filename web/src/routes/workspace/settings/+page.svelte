<script lang="ts">
 import WorkspaceHub from '$lib/components/WorkspaceHub.svelte';
 const groups=[
 {title:'Business settings',description:'Set up the business you trade as and the terms your team uses.',links:[
 ['Business setup','/workspace/onboarding','Business, authority and contact checks still outstanding.'],
 ['Default credit terms','/workspace/settings/credit-policy','Set default limits, payment periods and grace periods for new proposals.'],
 ['Team and access','/workspace/team','Invite staff and manage their permitted business actions.'],
 ['Receiving account','/workspace/settings/settlement','Choose and verify where your business receives payments.'],
 ['Kredit fees','/workspace/settings/billing','Review billing and platform fee arrangements.'],
 ['Business activity','/workspace/activity','Review the history of important business changes.']
 ]},
 {title:'Your personal account',description:'Shared across your business workspace and personal purchases.',links:[
 ['Account settings','/account','Your sign-in, what we send you, and how this device behaves.'],
 ['Supplier reminders','/account/suppliers','Manage permission for extra reminders from your connected suppliers.'],
 ['Introductions','/workspace/referral','Confirm an existing field-agent introduction, if one brought your business to Kredit.'],
 ['Field agent programme','/agents','Open the separate introduction and rewards programme.']
 ]}
 ] as {title:string;description:string;links:[string,string,string][]}[];
</script>
<WorkspaceHub title="Settings" description="Configure your business once. Keep personal account preferences in one place." {groups}/>
