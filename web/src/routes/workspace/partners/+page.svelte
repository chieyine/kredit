<script lang="ts">
 import WorkspaceHub from '$lib/components/WorkspaceHub.svelte';
 const groups=[
  {title:'Grow your customer network',description:'Manufacturers invite distributors. Distributors invite retailers. Every business owns its own customer relationships.',links:[
   ['Customers','/workspace/partners/customers','Open a business relationship, review its history or start another sale.'],
   ['Invite a business','/workspace/partners/customers/new','Send one private invitation. Your customer confirms its business before you trade.'],
   ['Import your network','/workspace/partners/import','Review a CSV of distributors or retailers and send invitations without duplicate entries.'],
   ['Invitation progress','/workspace/partners/invitations','See who has joined and where a first trade still needs to happen.']
  ]},
  {title:'Manage the other side of trade',description:'Your suppliers and personal customers have different journeys.',links:[
   ['Supplier purchases','/workspace/purchases','Your supplier agreements, what has been delivered, and what is still owed.'],
   ['Individual customers','/workspace/sales/consumers','Create and manage personal purchases for consumers, without business registration.']
  ]}
 ] as {title:string;description:string;links:[string,string,string][]}[];
</script>
<WorkspaceHub title="Partners" description="The businesses you buy from and the customers you sell to. Connect each relationship on its own terms." {groups}/>
